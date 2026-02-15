#!/usr/bin/env python3
"""
Generate three long-form test books (thriller, romance, fantasy) with Ollama.

Design goals:
- Produce at least N words per book (default: 40,000).
- Orchestrate chapter-by-chapter generation.
- Feed each chapter with:
  1) a running summary of story-so-far
  2) a predefined chapter plot objective
- Keep resumable state to survive interruptions/timeouts.
"""

from __future__ import annotations

import argparse
import json
import os
import re
import sys
import time
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import Any
from urllib import error, request


WORD_RE = re.compile(r"[A-Za-z0-9']+")


@dataclass
class BookSpec:
    genre: str
    title: str
    premise: str
    tone: str
    chapter_plans: list[str]


def now() -> str:
    return datetime.now().strftime("%H:%M:%S")


def log(msg: str) -> None:
    print(f"[{now()}] {msg}", flush=True)


def count_words(text: str) -> int:
    return len(WORD_RE.findall(text))


def post_json(url: str, payload: dict[str, Any], timeout_sec: int) -> dict[str, Any]:
    raw = json.dumps(payload).encode("utf-8")
    req = request.Request(url, data=raw, method="POST")
    req.add_header("Content-Type", "application/json")
    with request.urlopen(req, timeout=timeout_sec) as resp:
        body = resp.read().decode("utf-8", errors="replace")
    return json.loads(body)


def ollama_generate(
    endpoint: str,
    model: str,
    system: str,
    prompt: str,
    timeout_sec: int,
    temperature: float,
    seed: int | None,
    retries: int = 3,
) -> str:
    url = endpoint.rstrip("/") + "/api/generate"
    payload = {
        "model": model,
        "system": system,
        "prompt": prompt,
        "stream": False,
        "options": {
            "temperature": temperature,
            "top_p": 0.9,
            "num_predict": 4096,
        },
    }
    if seed is not None:
        payload["options"]["seed"] = seed

    attempt = 0
    while True:
        attempt += 1
        try:
            data = post_json(url, payload, timeout_sec=timeout_sec)
            response = str(data.get("response", "")).strip()
            if not response:
                raise RuntimeError("empty response from Ollama")
            return response
        except (error.URLError, TimeoutError, json.JSONDecodeError, RuntimeError) as exc:
            if attempt >= retries:
                raise RuntimeError(f"Ollama generation failed after {attempt} attempts: {exc}") from exc
            sleep_s = attempt * 2
            log(f"Retrying Ollama call ({attempt}/{retries}) after error: {exc}")
            time.sleep(sleep_s)


def check_ollama(endpoint: str, timeout_sec: int) -> None:
    url = endpoint.rstrip("/") + "/api/tags"
    req = request.Request(url, method="GET")
    try:
        with request.urlopen(req, timeout=timeout_sec) as resp:
            if resp.status < 200 or resp.status >= 300:
                raise RuntimeError(f"unexpected status: {resp.status}")
    except Exception as exc:
        raise RuntimeError(
            f"Cannot reach Ollama at {url}. Ensure service is running (e.g. `brew services start ollama`). Error: {exc}"
        ) from exc


def default_specs() -> list[BookSpec]:
    thriller_plans = [
        "Open with an engineered blackout during a high-security data transfer; protagonist Mara traces an impossible access signature.",
        "Mara discovers the breach implicates her missing brother; show conflicting evidence and a near-capture in a train terminal.",
        "A whistleblower gives Mara a dead-man switch drive and is assassinated minutes later.",
        "Mara decrypts part of the drive: a private intelligence consortium is staging market collapses.",
        "A failed handoff in Istanbul reveals a mole inside Mara's own unit.",
        "Mara and analyst Ren break into an archival vault and extract the ledger of covert payments.",
        "The mole frames Mara for treason; she goes off-grid and seeks help from an old rival.",
        "Flashback chapter: the brother's final operation and why he vanished.",
        "Mara runs a social-engineering sting to expose one shell company but triggers a citywide manhunt.",
        "Ren is kidnapped; captors demand the drive key and threaten timed leaks.",
        "Mara trades false keys and buys time while planting spyware in the captor network.",
        "A convoy ambush recovers Ren but confirms the enemy controls satellite surveillance windows.",
        "The pair infiltrate a closed summit where heads of finance and defense meet in secret.",
        "Mara broadcasts partial evidence; the leak causes panic but not arrests.",
        "Internal Affairs corners Mara; she escapes custody through a staged transfer crash.",
        "The brother contacts Mara through a one-use channel and names the mastermind: Director Vale.",
        "Mara verifies Vale manipulated both governments and criminal syndicates for deniable policy outcomes.",
        "A double-cross by Mara's rival destroys their safehouse and burns their remaining cash.",
        "Ren uncovers a physical kill-switch server farm tied to global trading algorithms.",
        "Team enters the server facility under storm cover; multiple squads converge.",
        "Mara confronts the brother, now coerced into running the system.",
        "They trigger a controlled shutdown but must choose between saving evidence and saving hostages.",
        "Final pursuit through flooded tunnels; Vale attempts escape with backup ledgers.",
        "Resolution: Vale exposed, brother's fate ambiguous, and Mara learns the network has successors.",
    ]
    romance_plans = [
        "Introduce Elena, a conservation architect, and Noah, a hotel developer, clashing over a historic seaside theater.",
        "City council grants temporary injunction; Elena and Noah must co-lead a feasibility study.",
        "Forced proximity: first site survey reveals hidden murals and Noah's genuine admiration for craft.",
        "Elena's ex resurfaces with funding leverage, complicating both project and trust.",
        "Noah confesses his company is under debt pressure and needs a fast outcome.",
        "A storm damages the theater; Elena and Noah spend the night protecting archives.",
        "They share personal losses that shaped their ambitions; emotional intimacy grows.",
        "Public hearing turns hostile after a leaked memo misrepresents Elena's plan.",
        "Noah takes blame publicly, damaging his board position but protecting Elena's reputation.",
        "Romantic midpoint: first kiss after restoring the main stage lights.",
        "Conflict rises: Elena learns Noah once approved demolition permits for a similar landmark.",
        "Elena withdraws, focusing on work; Noah begins independent restitution efforts.",
        "Secondary couple and friends challenge both leads to face fear rather than pride.",
        "Noah secures artisan partners and low-impact financing aligned with Elena's values.",
        "Elena's ex offers a perfect rescue deal that excludes Noah and displaces local artists.",
        "Elena chooses community-first terms but still refuses reconciliation with Noah.",
        "A gala fundraiser fails when anonymous donors withdraw under corporate pressure.",
        "Noah confronts his board and is removed as acting COO.",
        "Elena visits Noah's family inn, learning his mother once saved a heritage district.",
        "They collaborate on a revised plan balancing preservation, accessibility, and profitability.",
        "Final council vote is delayed after legal sabotage; they race to prove document fraud.",
        "Courtroom-adjacent climax: testimony from craftspeople and neighborhood elders secures injunction permanence.",
        "Grand gesture: Noah signs away controlling shares to protect theater covenants.",
        "Resolution: reopening night, committed partnership, and a future project together abroad.",
    ]
    fantasy_plans = [
        "Open in frostbound valley where apprentice mapmaker Ilya witnesses a starfall that awakens ancient runes.",
        "The village oracle reveals the starfall marks the return of the Hollow Crown.",
        "Ilya joins knight Ser Rowan and healer Tamsin on a quest to find the First Compass.",
        "They cross a mirror marsh where memories become physical threats.",
        "A band of sky-raiders attacks; Ilya discovers latent cartomancy magic.",
        "At ruined observatory they decode celestial routes to three ward-stones.",
        "Political chapter in capital: Regent Maelor seeks the Crown to cement rule.",
        "The party enters ember caverns and bargains with salamander scribes for passage.",
        "Tamsin's oath conflict: saving villagers versus staying on quest schedule.",
        "First ward-stone recovered, but Rowan is cursed with night-blindness.",
        "They seek a moon monastery where monks trade prophecies for true names.",
        "Ilya gives up family name to secure cure fragments and map insight.",
        "Sea crossing chapter with leviathan storm and mutiny among hired sailors.",
        "Second ward-stone lies in drowned library; retrieval costs a sacred relic.",
        "Maelor's hunters capture Tamsin; party forced into a rescue at glass fortress.",
        "Rescue succeeds, but Compass is stolen by an ally revealed as spy.",
        "The spy claims Crown can seal a greater abyss; moral ambiguity deepens.",
        "Ilya reconstructs lost map layer showing Crown's chamber beneath worldroot.",
        "Final journey through root-catacombs where time loops and paths rewrite themselves.",
        "Rowan overcomes curse by fighting without sight using star-echo techniques.",
        "Confrontation with Maelor and spy over whether to destroy or wield the Crown.",
        "Abyss breaches; team must fuse ward-stones while holding collapsing chamber.",
        "Ilya chooses to fracture Crown into three living oaths bound to companions.",
        "Resolution: kingdom enters uneasy peace; new order of map-keepers is founded.",
    ]
    return [
        BookSpec(
            genre="thriller",
            title="Null Meridian",
            premise="A cyber-intelligence officer uncovers a transnational conspiracy that engineers crises for profit and policy control.",
            tone="tense, procedural, cinematic but grounded",
            chapter_plans=thriller_plans,
        ),
        BookSpec(
            genre="romance",
            title="Theater of Tides",
            premise="Two professionals with opposing priorities must save a historic theater and confront what they truly value.",
            tone="emotionally layered, witty, intimate, heartfelt",
            chapter_plans=romance_plans,
        ),
        BookSpec(
            genre="fantasy",
            title="The Starcartographer's Oath",
            premise="A young mapmaker and unlikely allies race to prevent a relic from remaking the realm.",
            tone="epic, lyrical, adventurous with strong character bonds",
            chapter_plans=fantasy_plans,
        ),
    ]


def ensure_dir(path: Path) -> None:
    path.mkdir(parents=True, exist_ok=True)


def read_state(state_path: Path) -> dict[str, Any] | None:
    if not state_path.exists():
        return None
    return json.loads(state_path.read_text(encoding="utf-8"))


def write_state(state_path: Path, state: dict[str, Any]) -> None:
    tmp = state_path.with_suffix(".tmp")
    tmp.write_text(json.dumps(state, indent=2), encoding="utf-8")
    tmp.replace(state_path)


def summarize_chapter(
    endpoint: str,
    model: str,
    chapter_text: str,
    timeout_sec: int,
    seed: int | None,
) -> str:
    system = (
        "You summarize fiction chapters for continuity tracking. "
        "Output plain text, max 140 words, focused on plot events, character state, and unresolved threads."
    )
    prompt = (
        "Summarize this chapter for continuity memory.\n"
        "- Keep concrete facts.\n"
        "- Mention names, locations, stakes, and unresolved tensions.\n\n"
        "CHAPTER TEXT:\n"
        f"{chapter_text[:12000]}"
    )
    return ollama_generate(
        endpoint=endpoint,
        model=model,
        system=system,
        prompt=prompt,
        timeout_sec=timeout_sec,
        temperature=0.2,
        seed=seed,
    )


def merge_memory(
    endpoint: str,
    model: str,
    previous_memory: str,
    latest_summary: str,
    timeout_sec: int,
    seed: int | None,
) -> str:
    system = (
        "You maintain compact long-form story memory. "
        "Output plain text bullet points only, max 350 words."
    )
    prompt = (
        "Update the running story memory using the latest chapter summary.\n"
        "Preserve continuity facts, unresolved threads, alliances, betrayals, emotional state changes, and timeline markers.\n"
        "Drop fluff.\n\n"
        "PREVIOUS MEMORY:\n"
        f"{previous_memory}\n\n"
        "LATEST CHAPTER SUMMARY:\n"
        f"{latest_summary}\n\n"
        "Return updated memory now."
    )
    return ollama_generate(
        endpoint=endpoint,
        model=model,
        system=system,
        prompt=prompt,
        timeout_sec=timeout_sec,
        temperature=0.2,
        seed=seed,
    )


def make_chapter_prompt(
    spec: BookSpec,
    chapter_number: int,
    chapter_goal: str,
    story_memory: str,
    target_words: int,
) -> tuple[str, str]:
    system = (
        f"You are writing a {spec.genre} novel chapter-by-chapter. "
        "Write natural prose with varied sentence rhythm and pacing. "
        "Avoid repetitive phrasing and avoid reusing prior paragraphs verbatim. "
        "No meta commentary. No outlines. No markdown."
    )
    prompt = (
        f"Book title: {spec.title}\n"
        f"Genre: {spec.genre}\n"
        f"Premise: {spec.premise}\n"
        f"Tone: {spec.tone}\n"
        f"Chapter number: {chapter_number}\n"
        f"Target chapter length: about {target_words} words (minimum {max(1200, target_words - 250)}).\n\n"
        "Story summary so far:\n"
        f"{story_memory}\n\n"
        "General plot objective for this chapter:\n"
        f"{chapter_goal}\n\n"
        "Instructions:\n"
        "- Start with 'Chapter {N}: <title>' on the first line.\n"
        "- Continue directly with prose scenes.\n"
        "- Advance the story materially; include consequences from prior chapters.\n"
        "- End with a clear hook into the next chapter.\n"
        "- Output chapter text only.\n"
    ).replace("{N}", str(chapter_number))
    return system, prompt


def manuscript_text(title: str, chapters: list[str]) -> str:
    header = f"{title}\n\n"
    return header + "\n\n".join(chapters).strip() + "\n"


def generate_one_book(
    spec: BookSpec,
    output_dir: Path,
    endpoint: str,
    model: str,
    min_words: int,
    target_chapter_words: int,
    max_chapters: int,
    timeout_sec: int,
    seed: int | None,
) -> Path:
    slug = re.sub(r"[^a-z0-9]+", "_", spec.title.lower()).strip("_")
    state_path = output_dir / f"{slug}.state.json"
    book_path = output_dir / f"{slug}.txt"
    meta_path = output_dir / f"{slug}.meta.json"

    state = read_state(state_path) or {
        "title": spec.title,
        "genre": spec.genre,
        "chapters": [],
        "chapter_summaries": [],
        "story_memory": "No chapters yet.",
        "created_at": datetime.now().isoformat(),
    }
    chapters: list[str] = list(state.get("chapters", []))
    chapter_summaries: list[str] = list(state.get("chapter_summaries", []))
    story_memory: str = str(state.get("story_memory", "No chapters yet."))

    while count_words(manuscript_text(spec.title, chapters)) < min_words and len(chapters) < max_chapters:
        chapter_no = len(chapters) + 1
        if chapter_no <= len(spec.chapter_plans):
            chapter_goal = spec.chapter_plans[chapter_no - 1]
        else:
            chapter_goal = (
                "Continue escalation from prior chapter, deepen character consequences, "
                "and set up the final resolution without repeating earlier scenes."
            )

        log(f"{spec.genre.upper()} | generating chapter {chapter_no}")
        system, prompt = make_chapter_prompt(
            spec=spec,
            chapter_number=chapter_no,
            chapter_goal=chapter_goal,
            story_memory=story_memory,
            target_words=target_chapter_words,
        )
        chapter_text = ollama_generate(
            endpoint=endpoint,
            model=model,
            system=system,
            prompt=prompt,
            timeout_sec=timeout_sec,
            temperature=0.75,
            seed=seed,
        ).strip()

        if not chapter_text.lower().startswith("chapter "):
            chapter_text = f"Chapter {chapter_no}: Untitled\n\n{chapter_text}"

        ch_words = count_words(chapter_text)
        if ch_words < max(900, target_chapter_words // 2):
            log(
                f"{spec.genre.upper()} | chapter {chapter_no} too short ({ch_words} words), regenerating with stricter length request"
            )
            system_retry, prompt_retry = make_chapter_prompt(
                spec=spec,
                chapter_number=chapter_no,
                chapter_goal=chapter_goal + " Write fuller scenes and dialogue.",
                story_memory=story_memory,
                target_words=max(target_chapter_words, 2000),
            )
            chapter_text = ollama_generate(
                endpoint=endpoint,
                model=model,
                system=system_retry,
                prompt=prompt_retry,
                timeout_sec=timeout_sec,
                temperature=0.8,
                seed=seed,
            ).strip()
            ch_words = count_words(chapter_text)

        chapters.append(chapter_text)

        try:
            chapter_summary = summarize_chapter(
                endpoint=endpoint,
                model=model,
                chapter_text=chapter_text,
                timeout_sec=timeout_sec,
                seed=seed,
            ).strip()
        except Exception as exc:
            chapter_summary = f"Chapter {chapter_no} summary unavailable due to error: {exc}"

        chapter_summaries.append(chapter_summary)

        try:
            story_memory = merge_memory(
                endpoint=endpoint,
                model=model,
                previous_memory=story_memory,
                latest_summary=chapter_summary,
                timeout_sec=timeout_sec,
                seed=seed,
            ).strip()
        except Exception:
            tail = "\n".join(chapter_summaries[-5:])
            story_memory = f"Recent continuity notes:\n{tail}"

        state.update(
            {
                "chapters": chapters,
                "chapter_summaries": chapter_summaries,
                "story_memory": story_memory,
                "updated_at": datetime.now().isoformat(),
                "word_count": count_words(manuscript_text(spec.title, chapters)),
            }
        )
        write_state(state_path, state)

        manuscript = manuscript_text(spec.title, chapters)
        book_path.write_text(manuscript, encoding="utf-8")
        log(
            f"{spec.genre.upper()} | chapter {chapter_no} complete | chapter_words={ch_words} | total_words={count_words(manuscript)}"
        )

    manuscript = manuscript_text(spec.title, chapters)
    total_words = count_words(manuscript)
    metadata = {
        "title": spec.title,
        "genre": spec.genre,
        "model": model,
        "endpoint": endpoint,
        "total_words": total_words,
        "chapters": len(chapters),
        "target_min_words": min_words,
        "generated_at": datetime.now().isoformat(),
        "state_file": str(state_path),
        "chapter_summaries": chapter_summaries,
    }
    meta_path.write_text(json.dumps(metadata, indent=2), encoding="utf-8")

    if total_words < min_words:
        raise RuntimeError(
            f"{spec.title} finished at {total_words} words, below requested minimum {min_words}. "
            f"Increase --max-chapters or --target-chapter-words and rerun."
        )

    return book_path


def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="Generate long-form test books with Ollama.")
    p.add_argument("--model", default="llama3.1:8b", help="Ollama model name (default: llama3.1:8b)")
    p.add_argument("--endpoint", default="http://127.0.0.1:11434", help="Ollama base URL")
    p.add_argument("--output-dir", default="books/generated", help="Directory for generated manuscripts")
    p.add_argument("--min-words", type=int, default=40000, help="Minimum words per book")
    p.add_argument("--target-chapter-words", type=int, default=1800, help="Target words per generated chapter")
    p.add_argument("--max-chapters", type=int, default=32, help="Hard cap on chapters per book")
    p.add_argument("--timeout-sec", type=int, default=240, help="Timeout per Ollama call")
    p.add_argument("--seed", type=int, default=None, help="Optional seed for reproducibility")
    return p.parse_args()


def main() -> int:
    args = parse_args()
    output_dir = Path(args.output_dir).resolve()
    ensure_dir(output_dir)

    check_ollama(args.endpoint, timeout_sec=max(5, min(args.timeout_sec, 30)))
    log(f"Ollama reachable at {args.endpoint}")
    log(f"Output directory: {output_dir}")
    log(f"Model: {args.model}")

    generated: list[Path] = []
    for spec in default_specs():
        log(f"Starting book generation: {spec.title} ({spec.genre})")
        out_path = generate_one_book(
            spec=spec,
            output_dir=output_dir,
            endpoint=args.endpoint,
            model=args.model,
            min_words=args.min_words,
            target_chapter_words=args.target_chapter_words,
            max_chapters=args.max_chapters,
            timeout_sec=args.timeout_sec,
            seed=args.seed,
        )
        generated.append(out_path)
        log(f"Completed: {out_path}")

    log("All books generated successfully:")
    for pth in generated:
        words = count_words(pth.read_text(encoding="utf-8"))
        log(f"- {pth.name}: {words} words")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except KeyboardInterrupt:
        print("\nInterrupted by user.", file=sys.stderr)
        raise SystemExit(130)
