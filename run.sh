set -e

TAILWIND=frontend/styles/tailwindcss
INPUT_CSS=frontend/styles/input.css
OUTPUT_CSS=frontend/styles/style.css

"$TAILWIND" -i "$INPUT_CSS" -o "$OUTPUT_CSS"

DIR="./go_backend/"

if lsof -t -nP -iTCP:8080 -sTCP:LISTEN >/dev/null 2>&1; then
  echo "port 8080 is already in use; stop that process first."
  echo "example: lsof -t -nP -iTCP:8080 -sTCP:LISTEN | xargs kill"
  exit 1
fi

cd "$DIR"
go build .
go run .
