import { useEffect, useRef } from "react";
import { Timeline } from "vis-timeline/standalone";
import { DashboardData } from "../types";

type Props = { data: DashboardData };

export function StructureTab({ data }: Props) {
  const timelineRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!timelineRef.current || data.timeline.length === 0) {
      return;
    }
    const now = new Date();
    const items = data.timeline.map((evt, i) => ({
      id: i + 1,
      content: `<strong>${evt.time_marker}</strong><br/>${evt.event}`,
      start: new Date(now.getTime() + i * 24 * 60 * 60 * 1000),
    }));
    const tl = new Timeline(timelineRef.current, items, {
      height: "340px",
      zoomable: false,
      moveable: false,
      stack: true,
      showCurrentTime: false,
    });
    return () => tl.destroy();
  }, [data.timeline]);

  return (
    <section className="panel-grid">
      <article className="panel">
        <h2>Chronos Timeline</h2>
        <div ref={timelineRef} className="vis-host" />
      </article>
      <article className="panel">
        <h2>Save the Cat Beats</h2>
        <ul className="list">
          {data.beats.map((b) => (
            <li key={b.name}>
              <strong>{b.name}</strong> (Ch {b.startChapter}-{b.endChapter})<br />
              <span className="muted">{b.reasoning}</span>
            </li>
          ))}
        </ul>
        <h2>Chapter Coverage</h2>
        <p>{data.chapterCount} chapters scanned with per-chapter logs in the console.</p>
      </article>
    </section>
  );
}
