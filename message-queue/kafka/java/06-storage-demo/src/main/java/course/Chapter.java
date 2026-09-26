package course;

class Chapter {
  static final String TOPIC = "orders.c06";
  static final boolean COMPACT = true;
  static final String USAGE = "snapshot|tombstone";

  static void run(String command, App.Options o) throws Exception {
    if (command.equals("tombstone")) {
      try (var p =
          new org.apache.kafka.clients.producer.KafkaProducer<String, String>(
              App.producerConfig(o))) {
        String key = o.value("prefix", "order") + "-1";
        p.send(
                new org.apache.kafka.clients.producer.ProducerRecord<String, String>(
                    o.topic(), App.partition(key, p.partitionsFor(o.topic()).size()), key, null))
            .get();
      }
    } else if (command.equals("snapshot")) {
      var state = new java.util.TreeMap<String, App.Event>();
      App.consume(
          o,
          (e, r) -> {
            if (e.type().equals("Tombstone")) state.remove(e.order_id());
            else if (!state.containsKey(e.order_id())
                || e.order_version() >= state.get(e.order_id()).order_version())
              state.put(e.order_id(), e);
          });
      state.forEach(
          (key, e) ->
              System.out.printf(
                  "STATE order=%s type=%s version=%d%n", key, e.type(), e.order_version()));
    } else throw new IllegalArgumentException("unknown command: " + command);
  }
}
