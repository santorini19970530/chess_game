# Chess / Xiangqi / Shogi — AI play + coaching + diagram→FEN

Multi-variant board game with explainable AI coaching (CM3070 Final Project, University of London / Goldsmiths).

Three models in the product: **Fairy-Stockfish** (play/analysis), **Ollama** (text coach; heuristic fallback if off), **Chess_diagram_to_FEN** (image → FEN, confirm before load).

Play Chess, Xiangqi, or Shogi (human vs human, human vs AI, AI vs AI).

**Online demo (no install):** https://eunhachessgame.win and https://www.eunhachessgame.win — Fairy-Stockfish play works; coach text uses the heuristic fallback only (Ollama is not installed on the server because of disk space).

**Two ways to install:** [A — Docker](#a--install-with-docker) (any OS) · [B — without Docker](#b--install-without-docker) (macOS / Linux / Windows with Windows Subsystem for Linux).

---

## How to Play

1. Open http://localhost:8080
2. Pick game → mode → (for AI) strength → New Game
3. Move on the board (or type a command). AI replies in Human vs AI / AI vs AI.
4. Notes / win chance / coach text update after moves (heuristic if Ollama is off)
5. Preferred: diagram import (**confirm** before load). Also: load moves, clock, Preview, **?** help (top-right)

Moves: Chess `e2e4`; Xiangqi `a4a5` / `h3h10`; Shogi `c3c4`, promote `e8e9+`, drop `P*e5`.

---

## A — Install with Docker

Best for markers on **Windows, macOS, or Linux**. Needs **Git + Docker** only (no Go/Python/engine build on the host).

### Step 1. Install tools

| OS | Git | Docker |
|----|-----|--------|
| **Windows** | [Git for Windows](https://git-scm.com/download/win) | [Docker Desktop](https://docs.docker.com/desktop/setup/install/windows-install/) — start it (Windows Subsystem for Linux backend is fine) |
| **macOS** | Git | [Docker Desktop](https://docs.docker.com/desktop/setup/install/mac-install/) — start it |
| **Linux** | `sudo apt install -y git` (or distro equivalent) | e.g. `sudo apt install -y docker.io docker-compose-plugin`, then `sudo usermod -aG docker "$USER"` and **re-login** |

```bash
# check if git and docker has been installed and refreshed
git --version && docker version && docker compose version
```

### Step 2. Clone and start

Run the following commands to download the project, then build and start the backend and analyser:

```bash
git clone https://github.com/santorini19970530/chess_game.git
cd chess_game
docker compose up --build
```

Leave the terminal open while you use the app. Open **http://localhost:8080**. First build can take several minutes.

Stop: `Ctrl+C`. Detached: `docker compose up -d --build` / `docker compose down`.

**Quick check:** Chess → Human vs AI → Intermediate → one move → AI replies.

Step 2 alone is enough to play. Coach text uses a **heuristic** fallback when no language model is connected.

### Step 3. Language model coach (preferred, not required)

A local **Ollama** coach is preferred for plain-language notes. The game still runs without it (heuristic coach).

1. Install [Ollama](https://ollama.com/) on the host and pull a model, e.g. `ollama pull llama3.2`.
2. Start Compose with the language model enabled (instead of the plain command in Step 2):

```bash
# coach via host Ollama (install Ollama, pull model first)
LLM_PROVIDER=ollama OLLAMA_MODEL=llama3.2 docker compose up --build
```

| Env | Meaning |
|-----|---------|
| `LLM_PROVIDER=ollama` | Use host Ollama at `host.docker.internal:11434` |
| `OLLAMA_MODEL` | default `llama3.2` |
| `NNUE_HOST_DIR` | host folder with `nn-3475407dc199.nnue` (default `../_local_nnue`) |
| `CHESS_DIAGRAM_HOST_DIR` | vision vendor clone (default `../_local_Chess_diagram_to_FEN`) |

Missing NNUE folder is fine — play still works with classical eval.

### Step 4. Diagram import — Chess_diagram_to_FEN (preferred, not required)

This is the **third model** (image → FEN). Preferred so markers can try diagram upload → confirm → load. Without it, board play still works; you just cannot import from a picture.

1. Next to the clone (parent of `chess_game/`), clone the vendor and download its weights (see the vendor project README for `uv sync` / `download_models.sh`).
2. Folder name expected by Compose: `../_local_Chess_diagram_to_FEN` (or set `CHESS_DIAGRAM_HOST_DIR`).
3. Restart Compose (`docker compose up --build`). The analyser reads that folder for `/fen_from_image`.

macOS may need `brew install cairo`; Linux: `libcairo2-dev`.

---

## B — Install without Docker

Runs Go + Python + Fairy-Stockfish on the host.
Use **macOS**, **Linux**, or **Windows with Windows Subsystem for Linux**.
On plain Windows without that Linux environment, please use [A — Docker](#a--install-with-docker) instead.

### Step 1. Install tools

Install these if the version checks below fail:

| Tool | Windows (use Windows Subsystem for Linux) | macOS | Linux (Debian/Ubuntu example) |
|------|-------------------------------------------|-------|-------------------------------|
| **Git** | `sudo apt install -y git` inside the Linux environment | Xcode Command Line Tools, or https://git-scm.com/download/mac | `sudo apt install -y git` |
| **Go** 1.22+ | https://go.dev/dl/ (Linux build inside the Linux environment) | https://go.dev/dl/ | https://go.dev/dl/ or `sudo apt install -y golang-go` (check version ≥ 1.22) |
| **Python** 3.10+ with `pip` | `sudo apt install -y python3 python3-pip` | usually preinstalled; else https://www.python.org/downloads/ — use `python3` / `pip3` | `sudo apt install -y python3 python3-pip` |
| **C++ build tools** (`make`, `g++`) | `sudo apt install -y build-essential` | install Xcode Command Line Tools: `xcode-select --install` | `sudo apt install -y build-essential` |

Plain Windows without a Linux environment: use [A — Install with Docker](#a--install-with-docker) instead.

```bash
# check if the required installations are done
git --version && go version && python3 --version && make --version
```

If any command says “not found”, install that tool from the table, open a **new** terminal, and check again.

### Step 2. Clone the Project Repo

```bash
git clone https://github.com/santorini19970530/chess_game.git
cd chess_game
```

### Step 3. Python dependencies

Installs the Python packages listed in `py_analyser/requirements.txt` (Flask, python-chess, Pillow, and so on) so the analyser service can run.

```bash
cd py_analyser
python3 -m pip install -r requirements.txt
cd ..
```

### Step 4. Build Fairy-Stockfish (one-time)

Compile the engine binary once after cloning.
You do not need to rebuild every time you start the app (only again if you delete the binary or change the engine source).

```bash
cd py_analyser/Fairy-Stockfish-fairy_sf_14/src
make -j build ARCH=x86-64-modern    # Apple Silicon: try ARCH=apple-silicon or ARCH=armv8
# binary should be: ./stockfish
cd ../../..
```

`make help` lists architectures.
Or set `FAIRY_STOCKFISH_PATH` to any Fairy-Stockfish 14 binary.

**If Fairy-Stockfish does not build or work:**

| What you see | Likely cause | What to do |
|--------------|--------------|------------|
| `make: command not found` / `g++: not found` | C++ tools missing | Finish Step 1 (`build-essential` or Xcode Command Line Tools) |
| `make` errors about architecture / unknown `ARCH` | Wrong `ARCH=` for your CPU | Run `make help` and pick a matching `ARCH` (Apple Silicon often `apple-silicon` or `armv8`) |
| No `./stockfish` file after `make` | Build failed or wrong folder | Stay in `.../src`, re-run `make`, confirm the file exists: `ls -l stockfish` |
| Backend log: `fairy-stockfish unavailable` | Binary missing, wrong path, or not executable | Re-run Step 4, or `export FAIRY_STOCKFISH_PATH=/full/path/to/stockfish`, then restart |
| Python / analyser: set `FAIRY_STOCKFISH_PATH`… | Same — analyser cannot find the binary | Same fix as above |
| UI plays but AI is weak / no engine suggestions | `USE_FAIRY_STOCKFISH` not set, or engine failed and Go fell back | Start backend with `USE_FAIRY_STOCKFISH=true` (see Step 6); check terminal logs for Fairy-Stockfish errors |
| Notes say Fairy-Stockfish unavailable / evaluation unavailable | Analyser cannot start the engine | Fix path/binary; restart the analyser |

Still stuck: use [A — Install with Docker](#a--install-with-docker) (Fairy-Stockfish is built inside the image).

### Step 5. Diagram import — Chess_diagram_to_FEN (preferred, not required)

This is the **third model** (image → FEN). Preferred for the full three-model path. Without it, board play and Fairy-Stockfish AI still work; diagram upload will not.

1. From the parent of `chess_game/`:

```bash
cd ..
git clone https://github.com/tsoj/Chess_diagram_to_FEN.git _local_Chess_diagram_to_FEN
cd _local_Chess_diagram_to_FEN
# follow that repo: sync environment, download_models.sh, then install flask/Pillow/python-chess into that Python if needed
```

2. System library for the vision stack: macOS `brew install cairo`; Linux `sudo apt install -y libcairo2-dev`.
3. Point the analyser at it (or rely on the default sibling path):

```bash
export CHESS_DIAGRAM_TO_FEN_DIR="$(pwd)"   # while inside _local_Chess_diagram_to_FEN
```

4. When starting the analyser (Step 6), prefer that environment’s Python if it has torch/vision installed (same idea as `run.sh` using `../_local_Chess_diagram_to_FEN/.venv/bin/python`).

### Step 6. Start the game

**macOS (Apple Silicon):** if `./run.sh` works:

`run.sh` is a helper script that
(1) rebuilds frontend CSS with the bundled Tailwind tool,
(2) starts the Python analyser on port 8001 (Ollama coach if Ollama is already running, otherwise heuristic; uses the diagram vendor Python if that venv exists),
(3) starts the Go backend on port 8080 with Fairy-Stockfish enabled, and
(4) stops both when you press Ctrl+C.

```bash
cd chess_game   # if you were in the vendor folder
./run.sh
```

**Linux, Windows Subsystem for Linux, or if `run.sh` fails on Tailwind:** `style.css` is already in the repo — start the two services yourself:

```bash
# terminal 1 — python analyser
cd py_analyser
LLM_PROVIDER=heuristic python3 server.py
# if Ollama is running on :11434 you can use:
# LLM_PROVIDER=ollama OLLAMA_MODEL=llama3.2 python3 server.py
# if diagram vendor venv exists, use that python instead of python3

# terminal 2 — go backend server
cd go_backend
USE_FAIRY_STOCKFISH=true PY_ANALYSER_URL=http://127.0.0.1:8001 go run .
```

Open **http://localhost:8080** on browser to play the game.

### Optional extras (not required to play)

Steps 1–6 already cover play, Fairy-Stockfish, preferred diagram import, and heuristic coach. These are extra only:

- **Ollama:** preferred plain-language coach. Without it, notes use the heuristic fallback. Install Ollama, `ollama pull llama3.2`, start it before the analyser (or set `LLM_PROVIDER=ollama` as in the two-terminal example above).
- **NNUE file:** Chess neural eval weights (`nn-3475407dc199.nnue`). Without it, Fairy-Stockfish still plays with classical eval. Put the file in `../_local_nnue/` or set `FAIRY_STOCKFISH_NNUE_PATH`.

---

## Troubleshooting

| Problem | Fix |
|---------|-----|
| `docker` not found | Install/start Docker; Linux: re-login after `docker` group |
| Port 8080 / 8001 busy | Stop the other process |
| Docker build slow/fails | Free disk/RAM; retry `docker compose up --build` |
| `stockfish: no such file` | Finish Step 4 under install B, or set `FAIRY_STOCKFISH_PATH` |
| `./run.sh` / Tailwind error | Use the two-terminal start under Step 6 of install B (Start the game) |
| No LLM paragraphs | Normal without Ollama; enable Ollama or `LLM_PROVIDER=ollama` |
| Windows without Windows Subsystem for Linux | Use install **A** (Docker) |
