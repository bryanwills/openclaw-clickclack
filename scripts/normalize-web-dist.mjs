import { readdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";

const dist = process.argv[2];
if (!dist) {
  throw new Error("usage: node scripts/normalize-web-dist.mjs <dist>");
}

for (const file of ["index.html", "200.html"]) {
  trimTrailingWhitespace(join(dist, file));
}

for (const file of walk(join(dist, "_app", "immutable"))) {
  if (/\.(?:css|js|mjs)$/.test(file)) {
    trimTrailingWhitespace(file);
  }
}

function trimTrailingWhitespace(path) {
  const input = readFileSync(path, "utf8");
  const output = input.replace(/[ \t]+$/gm, "");
  if (output !== input) writeFileSync(path, output);
}

function* walk(path) {
  for (const entry of readdirSync(path, { withFileTypes: true })) {
    const child = join(path, entry.name);
    if (entry.isDirectory()) {
      yield* walk(child);
    } else {
      yield child;
    }
  }
}
