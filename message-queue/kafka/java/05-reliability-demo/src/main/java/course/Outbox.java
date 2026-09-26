package course;

class Outbox {
  static void run(String command, App.Options o) throws Exception {
    String url = System.getenv("MYSQL_URL");
    if (url == null) throw new IllegalArgumentException("set MYSQL_URL");
    try (var db =
        java.sql.DriverManager.getConnection(
            url, App.env("MYSQL_USER", "root"), App.env("MYSQL_PASSWORD", ""))) {
      if (command.equals("place")) {
        db.setAutoCommit(false);
        try {
          for (var e : App.events(o)) {
            if (e.order_version() == 1) {
              try (var s =
                  db.prepareStatement("INSERT INTO source_orders(order_id,status) VALUES (?,?)")) {
                s.setString(1, e.order_id());
                s.setString(2, e.type());
                s.executeUpdate();
              }
            } else {
              try (var s =
                  db.prepareStatement("UPDATE source_orders SET status=? WHERE order_id=?")) {
                s.setString(1, e.type());
                s.setString(2, e.order_id());
                s.executeUpdate();
              }
            }
            try (var s =
                db.prepareStatement(
                    "INSERT INTO order_outbox(event_id,topic,payload) VALUES (?,?,?)")) {
              s.setString(1, e.event_id());
              s.setString(2, o.topic());
              s.setString(3, App.JSON.writeValueAsString(e));
              s.executeUpdate();
            }
          }
          db.commit();
          System.out.println("OUTBOX_SAVED " + (2 * o.number("orders", 3)));
        } catch (Exception e) {
          db.rollback();
          throw e;
        }
      } else {
        var pending = new java.util.ArrayList<App.Event>();
        try (var s =
            db.prepareStatement(
                "SELECT payload FROM order_outbox WHERE topic=? AND published=0 ORDER BY id LIMIT"
                    + " 100")) {
          s.setString(1, o.topic());
          try (var r = s.executeQuery()) {
            while (r.next()) pending.add(App.decode(r.getString(1)));
          }
        }
        // 单投递进程；ACK 与数据库标记之间仍可能崩溃，不能宣称仅投递一次。
        try (var p =
            new org.apache.kafka.clients.producer.KafkaProducer<String, String>(
                App.producerConfig(o))) {
          for (var e : pending) {
            p.send(App.record(p, o, e)).get();
            if (App.env("FAIL_AFTER_SEND", "0").equals("1"))
              throw new IllegalStateException(
                  "injected failure after Kafka ACK; outbox remains pending");
            try (var s =
                db.prepareStatement("UPDATE order_outbox SET published=1 WHERE event_id=?")) {
              s.setString(1, e.event_id());
              s.executeUpdate();
            }
            System.out.println("RELAYED " + e.event_id());
          }
        }
      }
    }
  }
}
