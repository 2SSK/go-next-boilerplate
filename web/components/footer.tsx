import { site } from "@/config/site";

export function Footer() {
  return (
    <footer className="container-page mt-auto pt-16 pb-10">
      <div className="flex items-center justify-between gap-6 border-t pt-6">
        <p className="meta text-xs">{site.name}</p>
        <a href={site.repo} className="meta interactive hover:text-foreground text-xs">
          source
        </a>
      </div>
    </footer>
  );
}
