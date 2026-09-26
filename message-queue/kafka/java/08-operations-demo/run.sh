#!/bin/sh
set -eu
cd "$(dirname "$0")"
# Maven 和 java 使用同一个 JDK。
if [ -z "${JAVA_HOME:-}" ] && [ -d /opt/homebrew/opt/openjdk/libexec/openjdk.jdk/Contents/Home ]; then
  JAVA_HOME=/opt/homebrew/opt/openjdk/libexec/openjdk.jdk/Contents/Home
  export JAVA_HOME
fi
mvn -q -DskipTests package dependency:build-classpath -Dmdep.outputFile=target/classpath.txt
exec "${JAVA_HOME:+$JAVA_HOME/bin/}java" -Dorg.slf4j.simpleLogger.defaultLogLevel=warn -cp "target/classes:$(cat target/classpath.txt)" course.App "$@"
