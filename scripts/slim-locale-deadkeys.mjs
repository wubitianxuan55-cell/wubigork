#!/usr/bin/env node
/**
 * Locale dead-key audit for the Gaea frontend.
 *
 * Default is a dry run. Pass --write to remove the same dead-key set from
 * en/zh/zh-TW. The script adds no dependency and deliberately treats every
 * backtick dynamic translator prefix (for example t(`prefix.${id}`)) as alive.
 */
import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { fileURLToPath, pathToFileURL } from "node:url";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(scriptDir, "..");
const srcRoot = path.join(root, "frontend", "src");
const localeFiles = [
  { locale: "en", relative: "frontend/src/gaea/locales/en.ts" },
  { locale: "zh", relative: "frontend/src/gaea/locales/zh.ts", baseline: true },
  { locale: "zh-TW", relative: "frontend/src/gaea/locales/zh-TW.ts" },
];

const args = new Set(process.argv.slice(2));
const write = args.has("--write");
const jsonMode = args.has("--json");
const unknownArgs = [...args].filter((arg) => !["--write", "--json"].includes(arg));
if (unknownArgs.length > 0) {
  console.error(`Unknown argument(s): ${unknownArgs.join(", ")}`);
  process.exit(2);
}

function fail(message) {
  console.error(message);
  process.exit(1);
}

function parseLocale(file) {
  const source = fs.readFileSync(file.fullPath, "utf8");
  const lines = source.split(/\r?\n/);
  const entries = [];
  const seen = new Set();
  const pattern = /^[ \t]*(?:"((?:\\.|[^"\\])*)"|'((?:\\.|[^'\\])*)')[ \t]*:/;

  lines.forEach((line, index) => {
    const match = line.match(pattern);
    if (!match) return;
    const key = match[1] ?? match[2];
    if (seen.has(key)) fail(`Duplicate key ${JSON.stringify(key)} in ${file.relative}`);
    seen.add(key);
    entries.push({ key, lineIndex: index });
  });

  return { source, lines, entries, keys: seen };
}

function walk(dir, files = []) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      walk(fullPath, files);
      continue;
    }
    if (!/\.(?:ts|tsx)$/.test(entry.name)) continue;
    if (entry.name.includes(".test.") || entry.name.includes(".spec.")) continue;
    const relative = path.relative(root, fullPath).replaceAll("\\", "/");
    if (relative.startsWith("frontend/src/gaea/locales/")) continue;
    files.push({ fullPath, relative });
  }
  return files;
}

async function scanUsage(files, keys) {
  const exact = new Set();
  const dynamicPrefixes = new Map();
  const typescriptPath = path.join(root, "frontend", "node_modules", "typescript", "lib", "typescript.js");
  if (!fs.existsSync(typescriptPath)) {
    fail("TypeScript compiler not found at frontend/node_modules; run npm install before this audit.");
  }
  const ts = await import(pathToFileURL(typescriptPath));
  const isTranslatorName = (name) => /^(?:[A-Za-z_$][A-Za-z0-9_$]*T|t)$/.test(name);
  // Lexical fallback for direct translator templates and translator variables
  // initialized from a template. Restrict to dotted prefixes to avoid treating
  // unrelated short template prefixes as dictionary wildcards.
  const dynamicTemplatePattern = /`([^`]*?)\$\{/g;

  for (const file of files) {
    const source = fs.readFileSync(file.fullPath, "utf8");
    const sourceFile = ts.createSourceFile(file.fullPath, source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
    const variableTemplatePrefixes = new Map();
    const translatorVariableCalls = [];
    const addPrefix = (prefix, kind) => {
      if (prefix.length === 0 || ![...keys].some((key) => key.startsWith(prefix))) return;
      dynamicPrefixes.set(prefix, { prefix, source: file.relative, kind });
    };
    const collectTemplatePrefixes = (node, prefixes = []) => {
      if (ts.isTemplateExpression(node)) prefixes.push(node.head.text);
      node.forEachChild((child) => collectTemplatePrefixes(child, prefixes));
      return prefixes;
    };
    const visit = (node) => {
      if (ts.isStringLiteral(node) && keys.has(node.text)) exact.add(node.text);

      if (ts.isVariableDeclaration(node) && ts.isIdentifier(node.name) && node.initializer) {
        variableTemplatePrefixes.set(node.name.text, collectTemplatePrefixes(node.initializer));
      }

      if (ts.isCallExpression(node)) {
        const callee = node.expression.getText(sourceFile);
        const argument = node.arguments[0];
        if (argument !== undefined && isTranslatorName(callee)) {
          if (ts.isTemplateExpression(argument)) addPrefix(argument.head.text, "template-call");
          if (ts.isIdentifier(argument)) translatorVariableCalls.push(argument.text);
        }
      }

      node.forEachChild(visit);
    };
    visit(sourceFile);
    for (const variable of translatorVariableCalls) {
      for (const prefix of variableTemplatePrefixes.get(variable) ?? []) {
        addPrefix(prefix, "translator-variable");
      }
    }
    for (const match of source.matchAll(dynamicTemplatePattern)) {
      if (match[1].includes(".")) addPrefix(match[1], "template-expression");
    }
  }
  return { exact, dynamicPrefixes: [...dynamicPrefixes.values()].sort((a, b) => a.prefix.localeCompare(b.prefix)) };
}

const localeData = localeFiles.map((file) => {
  const fullPath = path.join(root, ...file.relative.split("/"));
  if (!fs.existsSync(fullPath)) fail(`Missing locale file: ${file.relative}`);
  return { ...file, fullPath, ...parseLocale({ ...file, fullPath }) };
});

const baseline = localeData.find((file) => file.baseline);
if (!baseline) fail("Baseline locale zh.ts was not configured.");
for (const file of localeData) {
  if (file === baseline) continue;
  const missing = [...baseline.keys].filter((key) => !file.keys.has(key));
  const extra = [...file.keys].filter((key) => !baseline.keys.has(key));
  if (missing.length > 0 || extra.length > 0) {
    fail(
      `Locale key sets are not synchronized for ${file.locale}: ` +
        `missing=${JSON.stringify(missing)} extra=${JSON.stringify(extra)}`,
    );
  }
}

const usage = await scanUsage(walk(srcRoot), baseline.keys);
const dynamicPrefixes = usage.dynamicPrefixes.map((item) => item.prefix);
const isAlive = (key) =>
  usage.exact.has(key) || dynamicPrefixes.some((prefix) => key.startsWith(prefix));
const deadKeys = baseline.entries.map((entry) => entry.key).filter((key) => !isAlive(key));

let result = {
  mode: write ? "write" : "dry-run",
  baseline: baseline.locale,
  keyCount: baseline.keys.size,
  staticUsedCount: usage.exact.size,
  dynamicPrefixes,
  dynamicProtectedCount: [...baseline.keys].filter(
    (key) => !usage.exact.has(key) && dynamicPrefixes.some((prefix) => key.startsWith(prefix)),
  ).length,
  deadKeyCount: deadKeys.length,
  deadKeys,
  files: localeData.map((file) => ({
    locale: file.locale,
    relative: file.relative,
    bytesBefore: Buffer.byteLength(file.source, "utf8"),
    bytesAfter: Buffer.byteLength(file.source, "utf8"),
  })),
};

if (write && deadKeys.length > 0) {
  for (const file of localeData) {
    const deadLines = new Set(file.entries.filter((entry) => deadKeys.includes(entry.key)).map((entry) => entry.lineIndex));
    const nextLines = file.lines.filter((_, index) => !deadLines.has(index));
    fs.writeFileSync(file.fullPath, nextLines.join(file.source.includes("\r\n") ? "\r\n" : "\n"), "utf8");
    result.files.find((summary) => summary.locale === file.locale).bytesAfter =
      Buffer.byteLength(fs.readFileSync(file.fullPath, "utf8"), "utf8");
  }
}

if (jsonMode) {
  console.log(JSON.stringify(result, null, 2));
} else {
  console.log(`mode: ${result.mode}`);
  console.log(`baseline: ${result.baseline}`);
  console.log(`keys: ${result.keyCount}`);
  console.log(`static used: ${result.staticUsedCount}`);
  console.log(`dynamic prefixes: ${dynamicPrefixes.length ? dynamicPrefixes.join(", ") : "(none)"}`);
  console.log(`dynamic protected: ${result.dynamicProtectedCount}`);
  console.log(`dead keys: ${result.deadKeyCount}`);
  if (deadKeys.length > 0) console.log(deadKeys.join("\n"));
  for (const file of result.files) {
    console.log(`${file.locale}: ${file.bytesBefore} -> ${file.bytesAfter} bytes`);
  }
}
