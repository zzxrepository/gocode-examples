package course;

class Chapter {
  static final String TOPIC = "orders.c07";
  static final boolean COMPACT = false;
  static final String USAGE = "aggregate|streams";

  static void run(String command, App.Options o) throws Exception {
    if (command.equals("aggregate")) {
      var windows = new java.util.TreeMap<java.time.Instant, long[]>();
      App.consume(
          o,
          (e, r) -> {
            if (!e.type().equals("OrderPaid")) return;
            var minute =
                java.time.Instant.parse(e.occurred_at())
                    .truncatedTo(java.time.temporal.ChronoUnit.MINUTES);
            var v = windows.computeIfAbsent(minute, k -> new long[2]);
            v[0]++;
            v[1] += e.amount_cents();
          });
      windows.forEach(
          (k, v) ->
              System.out.printf("WINDOW minute=%s count=%d amount_cents=%d%n", k, v[0], v[1]));
    } else if (command.equals("streams")) streams(o);
    else throw new IllegalArgumentException("unknown command: " + command);
  }

  static App.Event parse(String value) {
    try {
      return App.decode(value);
    } catch (Exception e) {
      throw new IllegalArgumentException(e);
    }
  }

  static void streams(App.Options o) throws Exception {
    var cfg = App.base(o);
    cfg.put("application.id", o.group());
    cfg.put("processing.guarantee", "exactly_once_v2");
    cfg.put("replication.factor", o.number("replicas", 1));
    cfg.put("state.dir", ".cache/streams");
    cfg.put("auto.offset.reset", "earliest");
    cfg.put("commit.interval.ms", 100);
    var builder = new org.apache.kafka.streams.StreamsBuilder();
    var strings = org.apache.kafka.common.serialization.Serdes.String();
    var consumed =
        org.apache.kafka.streams.kstream.Consumed.with(strings, strings)
            .withTimestampExtractor(
                (record, partitionTime) ->
                    java.time.Instant.parse(parse((String) record.value()).occurred_at())
                        .toEpochMilli());
    // 以事件时间划分一分钟窗口，允许 30 秒乱序；状态和 changelog 由 Streams 管理。
    builder.stream(o.topic(), consumed)
        .filter((key, value) -> parse(value).type().equals("OrderPaid"))
        .selectKey((key, value) -> "all")
        .groupByKey(org.apache.kafka.streams.kstream.Grouped.with(strings, strings))
        .windowedBy(
            org.apache.kafka.streams.kstream.TimeWindows.ofSizeAndGrace(
                java.time.Duration.ofMinutes(1), java.time.Duration.ofSeconds(30)))
        .aggregate(
            () -> "0:0",
            (key, value, total) -> {
              var parts = total.split(":");
              return (Long.parseLong(parts[0]) + 1)
                  + ":"
                  + (Long.parseLong(parts[1]) + parse(value).amount_cents());
            },
            org.apache.kafka.streams.kstream.Materialized
                .<String, String,
                    org.apache.kafka.streams.state.WindowStore<
                        org.apache.kafka.common.utils.Bytes, byte[]>>
                    as("paid-minute-totals")
                .withKeySerde(strings)
                .withValueSerde(strings))
        .toStream()
        .map(
            (key, value) ->
                org.apache.kafka.streams.KeyValue.pair(key.window().startTime().toString(), value))
        .to(
            o.topic() + ".paid-totals",
            org.apache.kafka.streams.kstream.Produced.with(strings, strings));
    try (var app = new org.apache.kafka.streams.KafkaStreams(builder.build(), cfg)) {
      var failure = new java.util.concurrent.atomic.AtomicReference<Throwable>();
      app.setUncaughtExceptionHandler(
          e -> {
            failure.set(e);
            return org.apache.kafka.streams.errors.StreamsUncaughtExceptionHandler
                .StreamThreadExceptionResponse.SHUTDOWN_CLIENT;
          });
      app.start();
      Thread.sleep(o.duration("timeout", "30s").toMillis());
      // 短时实验结束时主动离组，避免重启等待旧成员的会话过期。
      if (!app.close(
          org.apache.kafka.streams.CloseOptions.timeout(java.time.Duration.ofSeconds(10))
              .withGroupMembershipOperation(
                  org.apache.kafka.streams.CloseOptions.GroupMembershipOperation.LEAVE_GROUP))) {
        throw new IllegalStateException("streams close timed out");
      }
      if (failure.get() != null) throw new IllegalStateException("streams failed", failure.get());
      System.out.println("STREAMS output=" + o.topic() + ".paid-totals");
    }
  }
}
