package course;

import com.fasterxml.jackson.databind.ObjectMapper;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.time.Instant;
import java.util.*;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicReference;
import java.util.zip.CRC32;
import org.apache.kafka.clients.admin.*;
import org.apache.kafka.clients.consumer.*;
import org.apache.kafka.clients.producer.*;
import org.apache.kafka.common.TopicPartition;
import org.apache.kafka.common.errors.TopicExistsException;
import org.apache.kafka.common.serialization.StringDeserializer;
import org.apache.kafka.common.serialization.StringSerializer;

public class App {
  static final ObjectMapper JSON = new ObjectMapper();

  // 两个语言版本使用相同字段；金额单位为分。
  public record Event(
      int schema_version,
      String event_id,
      String order_id,
      String type,
      int order_version,
      long amount_cents,
      String occurred_at) {
    void validate() {
      if (schema_version != 1
          || event_id == null
          || event_id.isBlank()
          || order_id == null
          || order_id.isBlank()
          || amount_cents < 0)
        throw new IllegalArgumentException("invalid event identity/schema/amount");
      if (!(("OrderCreated".equals(type) && order_version == 1)
          || (("OrderPaid".equals(type) || "OrderCancelled".equals(type)) && order_version == 2)))
        throw new IllegalArgumentException("invalid event type/version");
      Instant.parse(occurred_at);
    }
  }

  static Event decode(String value) throws Exception {
    Event e = JSON.readValue(value, Event.class);
    e.validate();
    return e;
  }

  static int partition(String key, int size) {
    CRC32 crc = new CRC32();
    crc.update(key.getBytes(StandardCharsets.UTF_8));
    return (int) (crc.getValue() % size);
  }

  static final class Options {
    final Map<String, String> values = new HashMap<>();

    Options(String[] args) {
      Set<String> allowed =
          Set.of(
              "brokers",
              "topic",
              "group",
              "prefix",
              "mode",
              "orders",
              "max",
              "partitions",
              "replicas",
              "timeout",
              "delay",
              "duplicate",
              "abort");
      for (int i = 1; i < args.length; i++) {
        String k = args[i];
        if (!k.startsWith("--") || !allowed.contains(k.substring(2)))
          throw new IllegalArgumentException("unknown option: " + k);
        k = k.substring(2);
        if (k.equals("duplicate") || k.equals("abort")) values.put(k, "true");
        else {
          if (++i >= args.length) throw new IllegalArgumentException("missing value: " + k);
          values.put(k, args[i]);
        }
      }
      if (number("orders", 3) < 1
          || number("max", 6) < 0
          || number("partitions", 3) < 1
          || number("replicas", 1) < 1
          || duration("timeout", "30s").isNegative()
          || duration("timeout", "30s").isZero()
          || duration("delay", "0s").isNegative())
        throw new IllegalArgumentException("invalid option");
    }

    String value(String k, String fallback) {
      return values.getOrDefault(k, fallback);
    }

    int number(String k, int fallback) {
      return Integer.parseInt(value(k, Integer.toString(fallback)));
    }

    boolean enabled(String k) {
      return Boolean.parseBoolean(value(k, "false"));
    }

    String brokers() {
      return value("brokers", env("KAFKA_BROKERS", "127.0.0.1:9092"));
    }

    String topic() {
      return value("topic", env("KAFKA_TOPIC", Chapter.TOPIC));
    }

    String group() {
      return value("group", env("KAFKA_GROUP", Chapter.TOPIC + "-java"));
    }

    Duration duration(String k, String fallback) {
      String v = value(k, fallback);
      if (v.endsWith("ms"))
        return Duration.ofMillis(Long.parseLong(v.substring(0, v.length() - 2)));
      if (v.endsWith("s"))
        return Duration.ofSeconds(Long.parseLong(v.substring(0, v.length() - 1)));
      if (v.endsWith("m"))
        return Duration.ofMinutes(Long.parseLong(v.substring(0, v.length() - 1)));
      throw new IllegalArgumentException("duration needs ms/s/m suffix");
    }
  }

  static String env(String key, String fallback) {
    return System.getenv().getOrDefault(key, fallback);
  }

  static Properties base(Options o) {
    Properties p = new Properties();
    p.put("bootstrap.servers", o.brokers());
    p.put("client.id", "order-course-java");
    p.put("request.timeout.ms", "10000");
    p.put("default.api.timeout.ms", "15000");
    return p;
  }

  static Properties producerConfig(Options o) {
    Properties p = base(o);
    p.put("key.serializer", StringSerializer.class.getName());
    p.put("value.serializer", StringSerializer.class.getName());
    p.put("acks", "all");
    p.put("enable.idempotence", "true");
    p.put("max.in.flight.requests.per.connection", "1");
    p.put("compression.type", "snappy");
    p.put("linger.ms", "5");
    p.put("delivery.timeout.ms", "30000");
    p.put("max.block.ms", "15000");
    return p;
  }

  static Properties consumerConfig(Options o) {
    Properties p = base(o);
    p.put("key.deserializer", StringDeserializer.class.getName());
    p.put("value.deserializer", StringDeserializer.class.getName());
    p.put("group.id", o.group());
    p.put("group.protocol", "classic");
    p.put("partition.assignment.strategy", RangeAssignor.class.getName());
    p.put("enable.auto.commit", "false");
    p.put("auto.offset.reset", "earliest");
    p.put("isolation.level", "read_committed");
    return p;
  }

  static List<Event> events(Options o) {
    List<Event> result = new ArrayList<>();
    for (int i = 1; i <= o.number("orders", 3); i++) {
      String id = o.value("prefix", "order") + "-" + i;
      for (int v = 1; v <= 2; v++) {
        Event e =
            new Event(
                1,
                id + "-" + v,
                id,
                v == 1 ? "OrderCreated" : "OrderPaid",
                v,
                1000,
                Instant.now().toString());
        result.add(e);
        if (o.enabled("duplicate")) result.add(e);
      }
    }
    return result;
  }

  static ProducerRecord<String, String> record(KafkaProducer<String, String> p, Options o, Event e)
      throws Exception {
    e.validate();
    int n = p.partitionsFor(o.topic()).size();
    // 显式 CRC32 分区与 Go 保持一致；扩分区会改变映射。
    return new ProducerRecord<>(
        o.topic(), partition(e.order_id(), n), e.order_id(), JSON.writeValueAsString(e));
  }

  static void init(Options o) throws Exception {
    try (Admin a = Admin.create(base(o))) {
      NewTopic t =
          new NewTopic(o.topic(), o.number("partitions", 3), (short) o.number("replicas", 1));
      if (Chapter.COMPACT) t.configs(Map.of("cleanup.policy", "compact"));
      try {
        a.createTopics(List.of(t)).all().get(20, TimeUnit.SECONDS);
        System.out.println("created " + o.topic());
      } catch (java.util.concurrent.ExecutionException e) {
        if (e.getCause() instanceof TopicExistsException)
          System.out.println("topic already exists; existing settings unchanged");
        else throw e;
      }
    }
  }

  static void produce(Options o) throws Exception {
    String mode = o.value("mode", "sync");
    if (!mode.equals("sync") && !mode.equals("async"))
      throw new IllegalArgumentException("mode must be sync/async");
    try (KafkaProducer<String, String> p = new KafkaProducer<>(producerConfig(o))) {
      AtomicReference<Exception> failure = new AtomicReference<>();
      for (Event e : events(o)) {
        if (mode.equals("sync")) {
          RecordMetadata m = p.send(record(p, o, e)).get();
          System.out.printf(
              "ACK event=%s partition=%d offset=%d%n", e.event_id(), m.partition(), m.offset());
        } else
          p.send(
              record(p, o, e),
              (m, error) -> {
                if (error != null) failure.compareAndSet(null, error);
                else System.out.printf("ACK partition=%d offset=%d%n", m.partition(), m.offset());
              });
      }
      // 异步调用返回不代表 Broker 已确认；flush 后仍需检查回调错误。
      p.flush();
      if (failure.get() != null) throw failure.get();
    }
  }

  @FunctionalInterface
  interface Processor {
    void accept(Event e, ConsumerRecord<String, String> r) throws Exception;
  }

  static void consume(Options o, Processor processor) throws Exception {
    int count = 0, max = o.number("max", 6);
    long deadline = System.nanoTime() + o.duration("timeout", "30s").toNanos();
    try (KafkaConsumer<String, String> c = new KafkaConsumer<>(consumerConfig(o))) {
      c.subscribe(
          List.of(o.topic()),
          new ConsumerRebalanceListener() {
            public void onPartitionsRevoked(Collection<TopicPartition> p) {
              System.out.println("REVOKED " + p);
            }

            public void onPartitionsAssigned(Collection<TopicPartition> p) {
              System.out.println("ASSIGNED " + p);
            }
          });
      while (System.nanoTime() < deadline && (max == 0 || count < max)) {
        ConsumerRecords<String, String> batch = c.poll(Duration.ofMillis(250));
        for (ConsumerRecord<String, String> r : batch) {
          Event e;
          if (r.value() == null && Chapter.COMPACT)
            e = new Event(1, "", r.key(), "Tombstone", 0, 0, "");
          else e = decode(r.value());
          Thread.sleep(o.duration("delay", "0s").toMillis());
          processor.accept(e, r);
          // 只提交已处理记录的下一位置；不能把 poll 的整批位置提前提交。
          c.commitSync(
              Map.of(
                  new TopicPartition(r.topic(), r.partition()),
                  new OffsetAndMetadata(r.offset() + 1)));
          count++;
          if (max > 0 && count >= max) break;
        }
      }
    }
    if (max > 0 && count < max)
      throw new IllegalStateException("received " + count + "/" + max + " before timeout");
    System.out.println("CONSUMED " + count);
  }

  static void printEvent(Event e, ConsumerRecord<String, String> r) throws Exception {
    System.out.printf(
        "EVENT partition=%d offset=%d %s%n", r.partition(), r.offset(), JSON.writeValueAsString(e));
  }

  static void inspect(Options o) throws Exception {
    try (Admin a = Admin.create(base(o))) {
      TopicDescription d =
          a.describeTopics(List.of(o.topic())).allTopicNames().get().get(o.topic());
      Map<TopicPartition, OffsetSpec> start = new HashMap<>(), end = new HashMap<>();
      d.partitions()
          .forEach(
              p -> {
                TopicPartition tp = new TopicPartition(o.topic(), p.partition());
                start.put(tp, OffsetSpec.earliest());
                end.put(tp, OffsetSpec.latest());
              });
      var first = a.listOffsets(start).all().get();
      var last = a.listOffsets(end).all().get();
      for (var p : d.partitions()) {
        TopicPartition tp = new TopicPartition(o.topic(), p.partition());
        System.out.printf(
            "partition=%d leader=%d replicas=%s isr=%s start=%d end=%d%n",
            p.partition(),
            p.leader().id(),
            p.replicas().stream().map(n -> n.id()).toList(),
            p.isr().stream().map(n -> n.id()).toList(),
            first.get(tp).offset(),
            last.get(tp).offset());
      }
    }
  }

  public static void main(String[] args) {
    try {
      if (args.length == 0)
        throw new IllegalArgumentException("usage: init|produce|consume|inspect|" + Chapter.USAGE);
      Options o = new Options(args);
      switch (args[0]) {
        case "init" -> init(o);
        case "produce" -> produce(o);
        case "consume" -> consume(o, App::printEvent);
        case "inspect" -> inspect(o);
        default -> Chapter.run(args[0], o);
      }
    } catch (Exception e) {
      System.err.println("ERROR: " + e);
      System.exit(1);
    }
  }
}
