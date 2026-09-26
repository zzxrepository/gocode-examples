package course;

class Chapter {
  static final String TOPIC = "orders.c01";
  static final boolean COMPACT = false;
  static final String USAGE = "(no extra commands)";

  static void run(String command, App.Options o) {
    throw new IllegalArgumentException("unknown command: " + command);
  }
}
