import Link from "next/link";

type BrandProps = {
  href?: string;
  tag?: string;
};

export default function Brand({ href = "/", tag = "TEAMS" }: BrandProps) {
  const content = (
    <span className="brand">
      <span className="brand__dot" aria-hidden="true" />
      <span>SSHTHING</span>
      <span className="brand__slash">/</span>
      <span className="muted">{tag}</span>
    </span>
  );

  if (href) {
    return <Link href={href}>{content}</Link>;
  }
  return content;
}
