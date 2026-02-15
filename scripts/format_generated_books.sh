#!/usr/bin/env bash
set -euo pipefail

INPUT_DIR="${INPUT_DIR:-books/generated}"
OUTPUT_DIR="${OUTPUT_DIR:-books/formatted}"
PDF_ENGINE="${PDF_ENGINE:-auto}"
FONT_SIZE="${FONT_SIZE:-12pt}"
LINE_SPACING="${LINE_SPACING:-1.25}"
PAPER_SIZE="${PAPER_SIZE:-letter}"
MARGIN="${MARGIN:-1in}"

usage() {
  cat <<EOF
Usage: $0 [options]

Standardize generated manuscripts into comparable DOCX/PDF formats.

Options:
  --input-dir DIR        Source manuscript directory (default: ${INPUT_DIR})
  --output-dir DIR       Output directory (default: ${OUTPUT_DIR})
  --pdf-engine NAME      PDF engine: auto|xelatex|pdflatex|lualatex|tectonic|weasyprint|wkhtmltopdf
  --font-size SIZE       Font size for PDF output (default: ${FONT_SIZE})
  --line-spacing N       Line spacing for PDF output (default: ${LINE_SPACING})
  --paper-size NAME      letter|a4 (default: ${PAPER_SIZE})
  --margin SIZE          Page margin for PDF output (default: ${MARGIN})
  -h, --help             Show help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --input-dir) INPUT_DIR="$2"; shift 2 ;;
    --output-dir) OUTPUT_DIR="$2"; shift 2 ;;
    --pdf-engine) PDF_ENGINE="$2"; shift 2 ;;
    --font-size) FONT_SIZE="$2"; shift 2 ;;
    --line-spacing) LINE_SPACING="$2"; shift 2 ;;
    --paper-size) PAPER_SIZE="$2"; shift 2 ;;
    --margin) MARGIN="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown arg: $1" >&2; usage; exit 1 ;;
  esac
done

log() {
  printf '[%s] %s\n' "$(date '+%H:%M:%S')" "$*"
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Missing required command: $1" >&2
    exit 1
  }
}

detect_pdf_engine() {
  if [[ "$PDF_ENGINE" != "auto" ]]; then
    echo "$PDF_ENGINE"
    return 0
  fi
  local candidate
  for candidate in xelatex pdflatex lualatex tectonic weasyprint wkhtmltopdf; do
    if command -v "$candidate" >/dev/null 2>&1; then
      echo "$candidate"
      return 0
    fi
  done
  echo "none"
}

slug_to_title() {
  local stem="$1"
  local meta_file="${INPUT_DIR}/${stem}.meta.json"
  if [[ -f "$meta_file" ]]; then
    python3 - "$meta_file" <<'PY'
import json,sys
try:
    data=json.load(open(sys.argv[1], encoding="utf-8"))
    title=(data.get("title") or "").strip()
    if title:
        print(title)
        raise SystemExit(0)
except Exception:
    pass
raise SystemExit(1)
PY
    if [[ $? -eq 0 ]]; then
      return 0
    fi
  fi
  python3 - "$stem" <<'PY'
import sys
name = sys.argv[1].replace("_", " ").strip()
if not name:
    print("Untitled")
else:
    print(name[:1].upper() + name[1:])
PY
}

build_markdown_from_txt() {
  local input_txt="$1"
  local output_md="$2"
  local title="$3"
  python3 - "$input_txt" "$output_md" "$title" <<'PY'
import re, sys
from pathlib import Path

raw = Path(sys.argv[1]).read_text(encoding="utf-8", errors="replace")
dst = Path(sys.argv[2])
title = sys.argv[3].strip()

chapter_re = re.compile(r'^\s*chapter\s+([0-9ivxlcdm]+)\s*:\s*(.+?)\s*$', re.I)
log_blob_re = re.compile(
    r'\[\d{2}:\d{2}:\d{2}\]\s+(?:Ollama call (?:started|still running\.\.\.|completed)|Retrying Ollama call)[^\n\[]*',
    re.I
)
bare_log_re = re.compile(
    r'(?:Ollama call (?:started|still running\.\.\.|completed)[^\n]*|Retrying Ollama call[^\n]*)',
    re.I
)
embedded_log_sentence_re = re.compile(
    r'[^.!?\n]*ollama call[^.!?\n]*(?:attempt\s*\d+/\d+)?[^.!?\n]*[.!?]?',
    re.I
)
attempt_marker_re = re.compile(r'\battempt\s*\d+/\d+\b', re.I)
timestamp_line_re = re.compile(r'^\[\d{2}:\d{2}:\d{2}\]\s+', re.I)
word_re = re.compile(r"[A-Za-z0-9']+")

raw = log_blob_re.sub(" ", raw)
raw = bare_log_re.sub(" ", raw)
raw = embedded_log_sentence_re.sub(" ", raw)
raw = attempt_marker_re.sub(" ", raw)
src = raw.splitlines()

def norm(s):
    return re.sub(r'[^a-z0-9]+', ' ', s.lower()).strip()

def derive_chapter_title(lines, start_idx, chapter_num, book_title):
    j = start_idx
    n_book = norm(book_title)
    while j < len(lines):
        s = lines[j].strip()
        if not s:
            j += 1
            continue
        if chapter_re.match(s):
            break
        if timestamp_line_re.match(s):
            j += 1
            continue
        first_sentence = re.split(r'[.!?]', s, maxsplit=1)[0].strip(" -:\t")
        tokens = word_re.findall(first_sentence)
        if len(tokens) < 3:
            j += 1
            continue
        candidate = " ".join(tokens[:8]).strip()
        if not candidate:
            j += 1
            continue
        # Reject headings that just repeat the book title.
        if n_book and norm(candidate) == n_book:
            j += 1
            continue
        # Preserve natural casing from prose; only ensure sentence-style first letter.
        candidate = candidate[:1].upper() + candidate[1:]
        candidate = re.sub(r"\s+", " ", candidate).strip()
        if not candidate:
            j += 1
            continue
        return candidate
        j += 1
    return f"Chapter {chapter_num}"

def is_generic_chapter_title(ch_num, ch_title):
    t = ch_title.strip().lower()
    if t in {"untitled", "untitled chapter", "unknown", "tbd", ""}:
        return True
    num = str(ch_num).strip().lower()
    return t in {
        f"chapter {num}",
        f"ch {num}",
        f"ch. {num}",
    }

out = []
out.append("---")
out.append(f"title: \"{title.replace('\"', '')}\"")
out.append("lang: en-US")
out.append("---")
out.append("")

first_non_empty = ""
for line in src:
    if line.strip():
        first_non_empty = line.strip()
        break

start_idx = 0
if first_non_empty.lower() == title.lower():
    for i, line in enumerate(src):
        if line.strip():
            start_idx = i + 1
            break

idx = start_idx
last_emitted_ch_num = None
while idx < len(src):
    line = src[idx]
    stripped = line.strip()
    if not stripped:
        out.append("")
        idx += 1
        continue
    if timestamp_line_re.match(stripped):
        idx += 1
        continue
    m = chapter_re.match(stripped)
    if m:
        ch_num = m.group(1)
        ch_title = m.group(2).strip()
        # Skip placeholder headers when the next heading has same chapter number and a better title.
        if is_generic_chapter_title(ch_num, ch_title):
            skip_current_heading = False
            probe = idx + 1
            probe_limit = min(len(src), idx + 40)
            while probe < probe_limit:
                s2 = src[probe].strip()
                if not s2 or timestamp_line_re.match(s2):
                    probe += 1
                    continue
                m2 = chapter_re.match(s2)
                if m2:
                    if m2.group(1).strip().lower() == str(ch_num).strip().lower() and not is_generic_chapter_title(m2.group(1), m2.group(2)):
                        skip_current_heading = True
                        break
                    if m2.group(1).strip().lower() != str(ch_num).strip().lower():
                        break
                probe += 1
            if skip_current_heading:
                idx += 1
                continue
        if is_generic_chapter_title(ch_num, ch_title):
            ch_title = derive_chapter_title(src, idx + 1, ch_num, title)
        elif norm(ch_title) == norm(title):
            ch_title = derive_chapter_title(src, idx + 1, ch_num, title)
        # Skip duplicate chapter header for same chapter number.
        if last_emitted_ch_num is not None and str(last_emitted_ch_num).strip().lower() == str(ch_num).strip().lower():
            idx += 1
            continue
        out.append(f"# Chapter {ch_num}: {ch_title}")
        last_emitted_ch_num = ch_num
        out.append("")
        idx += 1
        continue
    out.append(stripped)
    idx += 1

dst.write_text("\n".join(out).rstrip() + "\n", encoding="utf-8")
PY
}

to_docx() {
  local md="$1"
  local docx="$2"
  pandoc "$md" \
    -f markdown \
    --standalone \
    --toc --toc-depth=2 \
    -o "$docx"
}

to_pdf_with_pandoc() {
  local md="$1"
  local pdf="$2"
  local engine="$3"
  pandoc "$md" \
    -f markdown \
    --standalone \
    --toc --toc-depth=2 \
    --pdf-engine="$engine" \
    -V "fontsize=${FONT_SIZE}" \
    -V "linestretch=${LINE_SPACING}" \
    -V "geometry:margin=${MARGIN}" \
    -V "papersize:${PAPER_SIZE}" \
    -o "$pdf"
}

to_pdf_with_wkhtmltopdf() {
  local md="$1"
  local pdf="$2"
  local html css
  html="$(mktemp /tmp/mhd_book_fmt_html.XXXXXX)"
  css="$(mktemp /tmp/mhd_book_fmt_css.XXXXXX)"
  cat >"$css" <<'CSS'
body {
  font-family: Georgia, "Times New Roman", serif;
  font-size: 12pt;
  line-height: 1.45;
  margin: 0;
}
h1 {
  page-break-before: always;
  font-size: 20pt;
  margin-top: 1.6em;
}
p {
  margin: 0 0 0.7em 0;
  text-align: justify;
}
CSS
  pandoc "$md" -f markdown --standalone --css "$css" -o "$html"
  wkhtmltopdf \
    --margin-top "${MARGIN}" \
    --margin-right "${MARGIN}" \
    --margin-bottom "${MARGIN}" \
    --margin-left "${MARGIN}" \
    "$html" "$pdf"
  rm -f "$html" "$css"
}

main() {
  require_cmd pandoc
  require_cmd python3

  mkdir -p "$OUTPUT_DIR"
  local engine
  engine="$(detect_pdf_engine)"
  if [[ "$engine" == "none" ]]; then
    log "No PDF engine detected. DOCX will be produced; PDF will be skipped."
    log "Install one of: xelatex, pdflatex, lualatex, tectonic, weasyprint, wkhtmltopdf."
  else
    log "Using PDF engine: ${engine}"
  fi

  local found=0
  local src base stem title tmp_md out_docx out_pdf
  while IFS= read -r src; do
    found=1
    base="$(basename "$src")"
    stem="${base%.txt}"
    title="$(slug_to_title "$stem")"
    tmp_md="$(mktemp /tmp/mhd_book_md.XXXXXX)"
    out_docx="${OUTPUT_DIR}/${stem}.docx"
    out_pdf="${OUTPUT_DIR}/${stem}.pdf"

    log "Formatting ${base}"
    build_markdown_from_txt "$src" "$tmp_md" "$title"

    to_docx "$tmp_md" "$out_docx"
    log "Wrote ${out_docx}"

    if [[ "$engine" == "wkhtmltopdf" ]]; then
      to_pdf_with_wkhtmltopdf "$tmp_md" "$out_pdf"
      log "Wrote ${out_pdf}"
    elif [[ "$engine" != "none" ]]; then
      if to_pdf_with_pandoc "$tmp_md" "$out_pdf" "$engine"; then
        log "Wrote ${out_pdf}"
      else
        log "PDF generation failed for ${base}; DOCX is available."
      fi
    fi

    rm -f "$tmp_md"
  done < <(
    find "$INPUT_DIR" -maxdepth 1 -type f -name '*.txt' \
      ! -name '*.memory.txt' \
      ! -name '*.summaries.txt' \
      | sort
  )

  if [[ "$found" -eq 0 ]]; then
    log "No manuscript .txt files found in ${INPUT_DIR}"
    exit 1
  fi

  log "Formatting complete. Output directory: ${OUTPUT_DIR}"
}

main "$@"
