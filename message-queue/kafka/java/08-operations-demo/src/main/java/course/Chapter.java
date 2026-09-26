package course;

class Chapter {
  static final String TOPIC = "orders.c08";
  static final boolean COMPACT = false;
  static final String USAGE = "lag";

  static void run(String command, App.Options o) throws Exception {
    if (!command.equals("lag")) throw new IllegalArgumentException("unknown command: " + command);
    try (var a = org.apache.kafka.clients.admin.Admin.create(App.base(o))) {
      var commits = a.listConsumerGroupOffsets(o.group()).partitionsToOffsetAndMetadata().get();
      var d = a.describeTopics(java.util.List.of(o.topic())).allTopicNames().get().get(o.topic());
      var queries =
          new java.util.HashMap<
              org.apache.kafka.common.TopicPartition, org.apache.kafka.clients.admin.OffsetSpec>();
      d.partitions()
          .forEach(
              p ->
                  queries.put(
                      new org.apache.kafka.common.TopicPartition(o.topic(), p.partition()),
                      org.apache.kafka.clients.admin.OffsetSpec.latest()));
      var ends = a.listOffsets(queries).all().get();
      ends.forEach(
          (p, end) -> {
            var c = commits.get(p);
            if (c == null)
              System.out.printf(
                  "partition=%d committed=none end=%d lag=unknown%n", p.partition(), end.offset());
            else
              System.out.printf(
                  "partition=%d committed=%d end=%d lag=%d%n",
                  p.partition(), c.offset(), end.offset(), end.offset() - c.offset());
          });
    }
  }
}
