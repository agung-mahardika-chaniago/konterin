#!/usr/bin/env node
/**
 * Pemeriksaan ringan dokumentasi Markdown:
 * - blok kode (```) seimbang
 * - tautan relatif ke file .md tidak putus
 *
 * Jalankan: node scripts/check-docs.mjs
 */
import { readdirSync, statSync, readFileSync, existsSync } from "node:fs";
import { join, resolve, dirname } from "node:path";

const root = process.cwd();
const files = [];

function walk(dir) {
  for (const entry of readdirSync(dir)) {
    if (entry === "node_modules" || entry.startsWith(".")) continue;
    const full = join(dir, entry);
    const info = statSync(full);
    if (info.isDirectory()) walk(full);
    else if (entry.endsWith(".md")) files.push(full);
  }
}

walk(root);

let issues = 0;

for (const file of files) {
  const text = readFileSync(file, "utf8");
  const lines = text.split(/\r?\n/);
  const fences = lines.filter((line) => line.startsWith("```")).length;

  if (fences % 2 !== 0) {
    console.log(`UNBALANCED FENCES: ${file} (${fences})`);
    issues++;
  }

  const links = text.matchAll(/\]\((docs\/[^)#]+|[^)#\s]+\.md)(#[^)]*)?\)/g);
  for (const match of links) {
    const target = resolve(dirname(file), match[1]);
    if (!existsSync(target)) {
      console.log(`BROKEN LINK: ${file} -> ${match[1]}`);
      issues++;
    }
  }
}

console.log(`Markdown files: ${files.length} | issues: ${issues}`);
process.exitCode = issues === 0 ? 0 : 1;
