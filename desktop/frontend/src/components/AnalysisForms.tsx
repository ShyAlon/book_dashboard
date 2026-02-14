import { FormEvent } from "react";

type Props = {
  excerpt: string;
  setExcerpt: (v: string) => void;
  filePath: string;
  setFilePath: (v: string) => void;
  loading: boolean;
  onAnalyzeExcerpt: (e: FormEvent) => void;
  onAnalyzeFile: (e: FormEvent) => void;
  onPickAndAnalyze: () => void;
};

export function AnalysisForms(props: Props) {
  return (
    <>
      <form className="analyze-form" onSubmit={props.onAnalyzeExcerpt}>
        <textarea value={props.excerpt} onChange={(e) => props.setExcerpt(e.target.value)} placeholder="Paste chapter excerpt and click Analyze Excerpt..." />
        <button type="submit" disabled={props.loading}>{props.loading ? "Analyzing..." : "Analyze Excerpt"}</button>
      </form>

      <form className="analyze-form file-form" onSubmit={props.onAnalyzeFile}>
        <input value={props.filePath} onChange={(e) => props.setFilePath(e.target.value)} placeholder="Absolute path to .docx/.pdf, then click Analyze File" />
        <button type="button" onClick={props.onPickAndAnalyze} disabled={props.loading} className="ghost">{props.loading ? "Analyzing..." : "Pick File..."}</button>
        <button type="submit" disabled={props.loading || props.filePath.trim() === ""}>{props.loading ? "Analyzing..." : "Analyze File"}</button>
      </form>
    </>
  );
}
