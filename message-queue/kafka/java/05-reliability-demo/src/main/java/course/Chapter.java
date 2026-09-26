package course;

class Chapter {
  static final String TOPIC = "orders.c05";
  static final boolean COMPACT = false;
  static final String USAGE = "project|transaction|place|relay";

  static void run(String command, App.Options o) throws Exception {
    switch (command) {
      case "place", "relay" -> Outbox.run(command, o);
      case "project" -> project(o);
      case "transaction" -> transaction(o);
      default -> throw new IllegalArgumentException("unknown command: " + command);
    }
  }

  static void transaction(App.Options o) throws Exception {
    var cfg = App.producerConfig(o);
    cfg.put("transactional.id", App.env("TRANSACTIONAL_ID", o.topic() + "-java-producer"));
    try (var p = new org.apache.kafka.clients.producer.KafkaProducer<String, String>(cfg)) {
      p.initTransactions();
      p.beginTransaction();
      try {
        for (var e : App.events(o)) p.send(App.record(p, o, e)).get();
        if (o.enabled("abort")) {
          p.abortTransaction();
          System.out.println("ABORTED");
        } else {
          p.commitTransaction();
          System.out.println("COMMITTED");
        }
      } catch (Exception e) {
        p.abortTransaction();
        throw e;
      }
    }
  }

  static void project(App.Options o) throws Exception {
    String url = System.getenv("MYSQL_URL");
    if (url == null) throw new IllegalArgumentException("set MYSQL_URL; run schema.sql first");
    try (var db =
        java.sql.DriverManager.getConnection(
            url, App.env("MYSQL_USER", "root"), App.env("MYSQL_PASSWORD", ""))) {
      db.setAutoCommit(false);
      App.consume(
          o,
          (e, r) -> {
            try {
              try (var s =
                  db.prepareStatement(
                      "INSERT INTO processed_events(consumer_name,event_id) VALUES (?,?)")) {
                s.setString(1, o.group());
                s.setString(2, e.event_id());
                s.executeUpdate();
              }
              // 去重记录和业务更新位于同一个数据库事务中。
              try (var s =
                  db.prepareStatement(
                      "INSERT INTO"
                          + " order_projection(consumer_name,order_id,status,version,amount_cents)"
                          + " VALUES (?,?,?,?,?) ON DUPLICATE KEY UPDATE"
                          + " status=IF(VALUES(version)>version,VALUES(status),status),amount_cents=IF(VALUES(version)>version,VALUES(amount_cents),amount_cents),version=GREATEST(version,VALUES(version))")) {
                s.setString(1, o.group());
                s.setString(2, e.order_id());
                s.setString(3, e.type());
                s.setInt(4, e.order_version());
                s.setLong(5, e.amount_cents());
                s.executeUpdate();
              }
              db.commit();
            } catch (java.sql.SQLException ex) {
              db.rollback();
              if (ex.getErrorCode() == 1062) {
                System.out.println("DUPLICATE " + e.event_id());
                return;
              }
              throw ex;
            }
            if (App.env("FAIL_AFTER_DB", "0").equals("1"))
              throw new IllegalStateException(
                  "injected failure after DB commit; offset not committed");
            System.out.println("APPLIED " + e.event_id());
          });
    }
  }
}
