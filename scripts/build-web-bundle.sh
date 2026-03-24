#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CLIENT_DIR="${DUEL_CLIENT_ROOT:-/Users/tahboubali/IdeaProjects/duel.io}"

TARGET_DIR="$ROOT_DIR/target"
CLASSES_DIR="$TARGET_DIR/classes"
WEB_APP_DIR="$ROOT_DIR/web/app"
LIB_DIR="$WEB_APP_DIR/lib"
LOCAL_DEP_DIR="$TARGET_DIR/deps"

mkdir -p "$CLASSES_DIR" "$LIB_DIR" "$LOCAL_DEP_DIR"

GSON_JAR="$HOME/.m2/repository/com/google/code/gson/gson/2.11.0/gson-2.11.0.jar"
WS_JAR="$HOME/.m2/repository/org/java-websocket/Java-WebSocket/1.6.0/Java-WebSocket-1.6.0.jar"
SLF4J_API_JAR="$HOME/.m2/repository/org/slf4j/slf4j-api/2.0.13/slf4j-api-2.0.13.jar"
SLF4J_NOP_JAR="$LOCAL_DEP_DIR/slf4j-nop-2.0.13.jar"

if [[ ! -d "$CLIENT_DIR/src/main/java" ]]; then
  echo "Java client sources not found at $CLIENT_DIR" >&2
  exit 1
fi

if [[ ! -f "$SLF4J_NOP_JAR" ]]; then
  curl -L "https://repo1.maven.org/maven2/org/slf4j/slf4j-nop/2.0.13/slf4j-nop-2.0.13.jar" -o "$SLF4J_NOP_JAR"
fi

javac --release 8 \
  -cp "$GSON_JAR:$WS_JAR:$SLF4J_API_JAR:$SLF4J_NOP_JAR" \
  -d "$CLASSES_DIR" \
  $(find "$CLIENT_DIR/src/main/java" -name '*.java' | sort)

jar --create --file "$WEB_APP_DIR/duel.io.jar" -C "$CLASSES_DIR" .

cp "$GSON_JAR" "$LIB_DIR/gson-2.11.0.jar"
cp "$WS_JAR" "$LIB_DIR/Java-WebSocket-1.6.0.jar"
cp "$SLF4J_API_JAR" "$LIB_DIR/slf4j-api-2.0.13.jar"
cp "$SLF4J_NOP_JAR" "$LIB_DIR/slf4j-nop-2.0.13.jar"

echo "Web bundle updated in $WEB_APP_DIR from $CLIENT_DIR"
