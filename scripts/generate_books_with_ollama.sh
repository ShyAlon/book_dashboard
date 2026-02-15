#!/usr/bin/env bash
set -euo pipefail

MODEL="${MODEL:-llama3.1:8b}"
ENDPOINT="${ENDPOINT:-http://127.0.0.1:11434}"
OUTPUT_DIR="${OUTPUT_DIR:-books/generated}"
MIN_WORDS="${MIN_WORDS:-40000}"
TARGET_CHAPTER_WORDS="${TARGET_CHAPTER_WORDS:-1800}"
MAX_CHAPTERS="${MAX_CHAPTERS:-32}"
TIMEOUT_SEC="${TIMEOUT_SEC:-240}"
SEED="${SEED:-}"
DEBUG_LOG="${DEBUG_LOG:-}"
MIN_CHAPTER_WORDS="${MIN_CHAPTER_WORDS:-}"

usage() {
  cat <<EOF
Usage: $0 [options]

Options:
  --model NAME                 Ollama model (default: ${MODEL})
  --endpoint URL               Ollama endpoint (default: ${ENDPOINT})
  --output-dir DIR             Output directory (default: ${OUTPUT_DIR})
  --min-words N                Minimum words per book (default: ${MIN_WORDS})
  --target-chapter-words N     Target words per chapter (default: ${TARGET_CHAPTER_WORDS})
  --max-chapters N             Max chapters per book (default: ${MAX_CHAPTERS})
  --timeout-sec N              Timeout per Ollama call (default: ${TIMEOUT_SEC})
  --min-chapter-words N        Accept chapter at/above this word count
  --seed N                     Optional deterministic seed
  -h, --help                   Show help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --model) MODEL="$2"; shift 2 ;;
    --endpoint) ENDPOINT="$2"; shift 2 ;;
    --output-dir) OUTPUT_DIR="$2"; shift 2 ;;
    --min-words) MIN_WORDS="$2"; shift 2 ;;
    --target-chapter-words) TARGET_CHAPTER_WORDS="$2"; shift 2 ;;
    --max-chapters) MAX_CHAPTERS="$2"; shift 2 ;;
    --timeout-sec) TIMEOUT_SEC="$2"; shift 2 ;;
    --min-chapter-words) MIN_CHAPTER_WORDS="$2"; shift 2 ;;
    --seed) SEED="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown argument: $1" >&2; usage; exit 1 ;;
  esac
done

log() {
  printf '[%s] %s\n' "$(date '+%H:%M:%S')" "$*" >&2
}

dbg() {
  local msg="$*"
  if [[ -n "${DEBUG_LOG}" ]]; then
    printf '[%s] %s\n' "$(date '+%H:%M:%S')" "$msg" >>"$DEBUG_LOG"
  fi
}

to_upper() {
  tr '[:lower:]' '[:upper:]' <<<"$1"
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Missing required command: $1" >&2
    exit 1
  }
}

slugify() {
  tr '[:upper:]' '[:lower:]' <<<"$1" | sed -E 's/[^a-z0-9]+/_/g; s/^_+|_+$//g'
}

word_count_file() {
  local file="$1"
  if [[ ! -f "$file" ]]; then
    echo 0
  else
    wc -w <"$file" | tr -d ' '
  fi
}

check_ollama() {
  curl -fsS --max-time 10 "${ENDPOINT%/}/api/tags" >/dev/null
}

ollama_generate() {
  local system="$1"
  local prompt="$2"
  local temperature="${3:-0.75}"
  local payload response status_code body_file curl_err_file

  payload="$(python3 - "$MODEL" "$system" "$prompt" "$temperature" "$SEED" <<'PY'
import json,sys
model,system,prompt,temp,seed = sys.argv[1:]
opts={"temperature":float(temp),"top_p":0.9,"num_predict":4096}
if seed not in ("", "None", "none"):
    opts["seed"]=int(seed)
print(json.dumps({
    "model": model,
    "system": system,
    "prompt": prompt,
    "stream": False,
    "options": opts
}))
PY
)"

  body_file="$(mktemp /tmp/mhd_ollama_body.XXXXXX)"
  curl_err_file="$(mktemp /tmp/mhd_ollama_curlerr.XXXXXX)"
  status_code="$(
    curl -sS --max-time "$TIMEOUT_SEC" \
      -o "$body_file" \
      -w "%{http_code}" \
      -H 'Content-Type: application/json' \
      -d "$payload" \
      "${ENDPOINT%/}/api/generate" 2>"$curl_err_file"
  )" || {
    echo "curl request failed: $(cat "$curl_err_file")" >&2
    dbg "curl request failed: $(cat "$curl_err_file")"
    rm -f "$body_file" "$curl_err_file"
    return 1
  }
  rm -f "$curl_err_file"
  response="$(cat "$body_file")"
  rm -f "$body_file"

  if [[ "$status_code" != "200" ]]; then
    echo "ollama returned HTTP $status_code body=$(printf '%s' "$response" | head -c 600)" >&2
    dbg "ollama returned HTTP $status_code body=$(printf '%s' "$response" | head -c 600)"
    return 1
  fi

  python3 -c '
import json,sys
raw = sys.argv[1]
try:
    data = json.loads(raw)
except Exception as exc:
    sys.stderr.write(f"json parse failed: {exc}; body={raw[:600]}\n")
    raise SystemExit(1)
text = (data.get("response") or "").strip()
if not text:
    sys.stderr.write(f"empty response field; body={raw[:600]}\n")
    raise SystemExit(1)
print(text)
' "$response"
}

ollama_generate_retry() {
  local system="$1"
  local prompt="$2"
  local temperature="${3:-0.75}"
  local attempt=1
  local max_attempts=3
  local heartbeat_sec=20
  local tmp_out tmp_err pid rc start_ts next_log now elapsed

  while (( attempt <= max_attempts )); do
    log "Ollama call started (attempt ${attempt}/${max_attempts}, temperature=${temperature})"
    dbg "ollama call start attempt=${attempt} temp=${temperature} prompt_chars=${#prompt}"
    tmp_out="$(mktemp /tmp/mhd_ollama_out.XXXXXX)"
    tmp_err="$(mktemp /tmp/mhd_ollama_err.XXXXXX)"
    (
      ollama_generate "$system" "$prompt" "$temperature"
    ) >"$tmp_out" 2>"$tmp_err" &
    pid=$!
    start_ts="$(date +%s)"
    next_log=$((start_ts + heartbeat_sec))

    while kill -0 "$pid" >/dev/null 2>&1; do
      sleep 2
      now="$(date +%s)"
      if (( now >= next_log )); then
        elapsed=$((now - start_ts))
        log "Ollama call still running... elapsed=${elapsed}s (attempt ${attempt}/${max_attempts})"
        next_log=$((next_log + heartbeat_sec))
      fi
    done

    wait "$pid" || rc=$?
    rc="${rc:-0}"
    if (( rc == 0 )) && [[ -s "$tmp_out" ]]; then
      elapsed=$(( $(date +%s) - start_ts ))
      log "Ollama call completed in ${elapsed}s"
      cat "$tmp_out"
      rm -f "$tmp_out" "$tmp_err"
      return 0
    fi
    elapsed=$(( $(date +%s) - start_ts ))
    log "Retrying Ollama call (${attempt}/${max_attempts}) after ${elapsed}s, error: $(cat "$tmp_err" 2>/dev/null || true)"
    dbg "ollama call failed attempt=${attempt} elapsed=${elapsed}s stderr=$(cat "$tmp_err" 2>/dev/null || true) stdout_snippet=$(head -c 300 "$tmp_out" 2>/dev/null || true)"
    rm -f "$tmp_out" "$tmp_err"
    sleep $((attempt * 2))
    attempt=$((attempt + 1))
    rc=0
  done

  echo "Ollama generation failed after retries" >&2
  return 1
}

chapter_summary() {
  local chapter_text="$1"
  local system prompt
  system="You summarize fiction chapters for continuity tracking. Output plain text under 140 words."
  prompt=$'Summarize the chapter for continuity memory.\nMention names, locations, decisions, and unresolved tensions.\n\nCHAPTER:\n'"${chapter_text:0:12000}"
  ollama_generate_retry "$system" "$prompt" "0.2"
}

merge_memory() {
  local previous_memory="$1"
  local latest_summary="$2"
  local system prompt
  system="You maintain compact long-form story memory. Output plain text bullet points under 350 words."
  prompt=$'Update story memory with the latest chapter summary.\nKeep continuity facts and unresolved threads.\n\nPREVIOUS MEMORY:\n'"$previous_memory"$'\n\nLATEST SUMMARY:\n'"$latest_summary"
  ollama_generate_retry "$system" "$prompt" "0.2"
}

write_manuscript() {
  local title="$1"
  local chapters_dir="$2"
  local book_file="$3"
  {
    printf '%s\n\n' "$title"
    local first=1
    local f
    while IFS= read -r f; do
      if (( first == 0 )); then
        printf '\n\n'
      fi
      first=0
      cat "$f"
    done < <(find "$chapters_dir" -type f -name '*.txt' | sort)
    printf '\n'
  } >"$book_file"
}

get_plan_block() {
  local genre="$1"
  case "$genre" in
    thriller)
      cat <<'EOF'
Open with an engineered blackout during a high-security transfer; Mara traces an impossible access signature.
Mara discovers the breach implicates her missing brother; conflict and near-capture.
A whistleblower gives Mara a dead-man-switch drive and is killed.
Partial decryption reveals a private network staging market collapses.
An operation fails, exposing a mole inside Mara's unit.
Mara and Ren steal ledgers from an archive vault.
Mara is framed for treason and goes off-grid.
Flashback: the brother's final mission and disappearance.
Mara runs a sting against a shell company; citywide manhunt begins.
Ren is kidnapped; captors demand the key.
Mara trades false keys while planting spyware.
Convoy ambush recovers Ren; enemy controls surveillance windows.
Team infiltrates a secret summit of power brokers.
Mara leaks evidence; panic spreads but no arrests.
Internal Affairs corners Mara; she escapes custody.
The brother contacts Mara and names Director Vale.
Mara proves Vale manipulates state and criminal actors.
A double-cross destroys the safehouse.
Ren identifies a kill-switch server farm.
Infiltration of server facility during storm cover.
Mara confronts her coerced brother.
They choose between saving evidence and hostages.
Final tunnel pursuit as Vale attempts escape.
Resolution with exposure, losses, and hints of successor networks.
EOF
      ;;
    romance)
      cat <<'EOF'
Introduce Elena and Noah clashing over a historic seaside theater.
Council grants injunction; they must co-lead a feasibility study.
Forced proximity reveals hidden murals and unexpected respect.
Elena's ex resurfaces with funding leverage.
Noah reveals debt pressure and urgency.
Storm damages theater; they protect archives overnight.
They share losses and deepen emotional intimacy.
Public hearing turns hostile after a leaked memo.
Noah takes public blame to protect Elena.
Romantic midpoint: first kiss after restoring stage lights.
Elena learns Noah once approved a prior demolition.
Elena withdraws; Noah starts restitution efforts.
Friends challenge both to face fear and pride.
Noah secures artisan-backed financing aligned with preservation.
Elena is offered a perfect deal that displaces local artists.
Elena chooses community terms but refuses reconciliation.
Fundraiser fails under corporate pressure.
Noah confronts board and is removed as acting COO.
Elena learns Noah's family legacy of preservation.
They co-design a balanced plan and rebuild trust.
Final vote delayed by legal sabotage.
Climax testimony secures permanent protection.
Grand gesture: Noah signs away control to protect covenants.
Resolution: reopening night and committed partnership.
EOF
      ;;
    fantasy)
      cat <<'EOF'
Ilya sees a starfall that awakens ancient runes.
Oracle reveals return of the Hollow Crown.
Ilya joins Rowan and Tamsin to find the First Compass.
They cross a mirror marsh of weaponized memories.
Sky-raiders attack; Ilya discovers cartomancy magic.
At a ruined observatory they decode routes to ward-stones.
Capital politics: Regent Maelor seeks the Crown.
In ember caverns they bargain with salamander scribes.
Tamsin faces oath conflict between duty and compassion.
First ward-stone recovered; Rowan is cursed.
At moon monastery they trade truths for prophecy.
Ilya gives up his family name for map insight.
Sea crossing with leviathan storm and mutiny.
Second ward-stone lies in a drowned library.
Maelor's hunters capture Tamsin; rescue mission begins.
Rescue succeeds but an ally steals the Compass.
Spy claims the Crown is needed to seal a greater abyss.
Ilya reconstructs hidden map layer beneath worldroot.
Final journey through root-catacombs with time distortions.
Rowan masters blind combat through star-echo forms.
Confrontation over whether to destroy or wield the Crown.
Abyss breach forces impossible cooperation.
Ilya fractures Crown into living oaths.
Resolution: uneasy peace and founding of a new order.
EOF
      ;;
    *)
      return 1
      ;;
  esac
}

generate_book() {
  local genre="$1"
  local title="$2"
  local premise="$3"
  local tone="$4"
  local slug chapters_dir book_file mem_file summaries_file meta_file
  local current_words chapter_no chapter_goal system prompt chapter_text ch_words summary merged genre_tag chapter_head min_chapter_words extend_words
  local -a plans=()

  slug="$(slugify "$title")"
  genre_tag="$(to_upper "$genre")"
  if [[ -n "$MIN_CHAPTER_WORDS" ]]; then
    min_chapter_words="$MIN_CHAPTER_WORDS"
  else
    min_chapter_words=$((TARGET_CHAPTER_WORDS * 70 / 100))
    if (( min_chapter_words < 900 )); then
      min_chapter_words=900
    fi
  fi
  chapters_dir="${OUTPUT_DIR}/${slug}_chapters"
  book_file="${OUTPUT_DIR}/${slug}.txt"
  mem_file="${OUTPUT_DIR}/${slug}.memory.txt"
  summaries_file="${OUTPUT_DIR}/${slug}.summaries.txt"
  meta_file="${OUTPUT_DIR}/${slug}.meta.json"

  mkdir -p "$chapters_dir"
  [[ -f "$mem_file" ]] || echo "No chapters yet." >"$mem_file"
  [[ -f "$summaries_file" ]] || : >"$summaries_file"

  while IFS= read -r line; do
    plans+=("$line")
  done < <(get_plan_block "$genre")
  write_manuscript "$title" "$chapters_dir" "$book_file"
  current_words="$(word_count_file "$book_file")"

  while (( current_words < MIN_WORDS )); do
    chapter_no="$(find "$chapters_dir" -type f -name '*.txt' | wc -l | tr -d ' ')"
    chapter_no=$((chapter_no + 1))
    if (( chapter_no > MAX_CHAPTERS )); then
      echo "${title} stopped at ${current_words} words (< ${MIN_WORDS}) and hit max chapters (${MAX_CHAPTERS})." >&2
      return 1
    fi

    if (( chapter_no <= ${#plans[@]} )); then
      chapter_goal="${plans[$((chapter_no - 1))]}"
    else
      chapter_goal="Continue escalation from prior chapter, deepen consequences, and set up future chapters without repeating scenes."
    fi

    log "${genre_tag} | generating chapter ${chapter_no}"
    system="You are writing a ${genre} novel chapter-by-chapter. Write natural prose with varied rhythm. Avoid repeating earlier paragraphs. No markdown."
    prompt=$'Book title: '"$title"$'\nGenre: '"$genre"$'\nPremise: '"$premise"$'\nTone: '"$tone"$'\nChapter number: '"$chapter_no"$'\nTarget chapter length: about '"$TARGET_CHAPTER_WORDS"$' words.\nHard minimum chapter length: '"$min_chapter_words"$' words.\n\nStory summary so far:\n'"$(cat "$mem_file")"$'\n\nGeneral plot objective for this chapter:\n'"$chapter_goal"$'\n\nInstructions:\n- Start with Chapter '"$chapter_no"$': <title>\n- Continue with prose only\n- Advance continuity materially\n- End on a hook into the next chapter\n- Do not stop early; fulfill the minimum length.'

    chapter_text="$(ollama_generate_retry "$system" "$prompt" "0.75")"
    chapter_head="$(printf '%s' "$chapter_text" | head -c 32 | tr '[:upper:]' '[:lower:]')"
    if [[ "$chapter_head" != chapter* ]]; then
      chapter_text=$'Chapter '"$chapter_no"$': Untitled\n\n'"$chapter_text"
    fi

    ch_words="$(python3 -c "import re,sys; print(len(re.findall(r\"[A-Za-z0-9']+\", sys.argv[1])))" "$chapter_text")"
    if (( ch_words < min_chapter_words )); then
      extend_words=$((TARGET_CHAPTER_WORDS - ch_words))
      if (( extend_words < 350 )); then
        extend_words=350
      fi
      log "${genre_tag} | chapter ${chapter_no} short (${ch_words} words), extending by ~${extend_words} words"
      chapter_text+=$'\n\n'"$(ollama_generate_retry \
        "You are extending an existing novel chapter. Continue immediately from the last line with the same style and continuity. No recap, no restart, no headings." \
        $'Continue this chapter naturally and add about '"$extend_words"$' more words before ending on a forward hook.\n\nCHAPTER TEXT SO FAR:\n'"${chapter_text:0:18000}" \
        "0.7")"
      ch_words="$(python3 -c "import re,sys; print(len(re.findall(r\"[A-Za-z0-9']+\", sys.argv[1])))" "$chapter_text")"
      log "${genre_tag} | chapter ${chapter_no} length after extension: ${ch_words} words"
    fi

    printf '%s\n' "$chapter_text" >"${chapters_dir}/chapter_$(printf '%03d' "$chapter_no").txt"

    if summary="$(chapter_summary "$chapter_text" 2>/dev/null)"; then
      :
    else
      summary="Chapter ${chapter_no}: summary unavailable."
    fi
    printf 'Chapter %d summary: %s\n' "$chapter_no" "$summary" >>"$summaries_file"

    if merged="$(merge_memory "$(cat "$mem_file")" "$summary" 2>/dev/null)"; then
      printf '%s\n' "$merged" >"$mem_file"
    else
      tail -n 24 "$summaries_file" >"$mem_file"
    fi

    write_manuscript "$title" "$chapters_dir" "$book_file"
    current_words="$(word_count_file "$book_file")"
    log "${genre_tag} | chapter ${chapter_no} complete | total_words=${current_words}"
  done

  python3 - "$title" "$genre" "$MODEL" "$ENDPOINT" "$current_words" "$MIN_WORDS" "$book_file" >"$meta_file" <<'PY'
import json,sys,datetime
title,genre,model,endpoint,total_words,min_words,book_file = sys.argv[1:]
print(json.dumps({
  "title": title,
  "genre": genre,
  "model": model,
  "endpoint": endpoint,
  "total_words": int(total_words),
  "target_min_words": int(min_words),
  "book_file": book_file,
  "generated_at": datetime.datetime.now().isoformat()
}, indent=2))
PY
  log "${genre_tag} | completed ${book_file} (${current_words} words)"
}

main() {
  require_cmd curl
  require_cmd python3
  mkdir -p "$OUTPUT_DIR"
  if [[ -z "$DEBUG_LOG" ]]; then
    DEBUG_LOG="${OUTPUT_DIR}/mhd-bookgen-debug-$(date '+%Y%m%d-%H%M%S').log"
  fi

  log "Checking Ollama at ${ENDPOINT}"
  check_ollama
  log "Ollama reachable. Model=${MODEL}"
  log "Output directory: ${OUTPUT_DIR}"
  log "Debug log: ${DEBUG_LOG}"
  dbg "startup endpoint=${ENDPOINT} model=${MODEL} output_dir=${OUTPUT_DIR} min_words=${MIN_WORDS} target_chapter_words=${TARGET_CHAPTER_WORDS} max_chapters=${MAX_CHAPTERS}"

  generate_book \
    "thriller" \
    "Null Meridian" \
    "A cyber-intelligence officer uncovers a transnational conspiracy engineering crises for profit and policy control." \
    "tense, procedural, cinematic but grounded"

  generate_book \
    "romance" \
    "Theater of Tides" \
    "Two professionals with opposing priorities must save a historic seaside theater and confront what they truly value." \
    "emotionally layered, witty, intimate, heartfelt"

  generate_book \
    "fantasy" \
    "The Starcartographer's Oath" \
    "A young mapmaker and unlikely allies race to prevent a relic from remaking the realm." \
    "epic, lyrical, adventurous with strong character bonds"

  log "All books generated."
}

main "$@"
