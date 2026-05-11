import { readFile, readdir, stat } from "node:fs/promises";
import path from "node:path";
import process from "node:process";

const appRoot = path.resolve(process.cwd());
const srcRoot = path.join(appRoot, "src");
const indexHtmlPath = path.join(appRoot, "index.html");

async function main() {
  const indexHtml = await readFile(indexHtmlPath, "utf8");
  assert(indexHtml.includes('/src/boot/main.js'), "index.html must boot from /src/boot/main.js");
  assert(!indexHtml.includes('/assets/app/main.js'), "index.html must not boot from /assets/app/main.js");
  for (const label of ["总览", "交付网络", "集群工作台", "机器工作台", "系统治理"]) {
    assert(indexHtml.includes(`>${label}<`), `index.html must include primary nav label: ${label}`);
  }
  for (const id of [
    "overview-ops-grid",
    "overview-activity-grid",
    "delivery-page-header",
    "clusters-page-header",
    "machine-page-header",
    "system-page-header",
    "clusters-workbench-grid",
    "clusters-insight-rail",
    "machine-operations-grid",
    "machine-activity-rail",
    "cluster-list-stage",
    "machine-quick-strip",
    "delivery-stage-strip",
    "system-governance-strip",
  ]) {
    assert(indexHtml.includes(`id="${id}"`), `index.html must include redesigned shell anchor: ${id}`);
  }

  await assertPathExists(srcRoot, "src directory must exist");
  for (const dir of ["boot", "core", "domains", "pages", "shared"]) {
    await assertPathExists(path.join(srcRoot, dir), `src/${dir} directory must exist`);
  }
  for (const dir of ["login", "overview", "delivery", "clusters", "machines", "system"]) {
    await assertPathExists(path.join(srcRoot, "pages", dir), `src/pages/${dir} directory must exist`);
  }
  await assertPathExists(path.join(srcRoot, "domains", "machines"), "src/domains/machines directory must exist");
  await assertPathExists(path.join(srcRoot, "domains", "delivery"), "src/domains/delivery directory must exist");
  await assertPathExists(path.join(srcRoot, "domains", "clusters"), "src/domains/clusters directory must exist");

  const files = await collectFiles(srcRoot);
  const jsFiles = files.filter((file) => file.endsWith(".js"));
  assert(jsFiles.length > 0, "src must contain JavaScript modules");

  for (const file of jsFiles) {
    const content = await readFile(file, "utf8");
    const imports = Array.from(content.matchAll(/import\s+(?:[^'"]+from\s+)?["']([^"']+)["']/g));
    for (const [, specifier] of imports) {
      if (!specifier.startsWith(".")) continue;
      const resolved = path.resolve(path.dirname(file), specifier);
      await assertModuleTarget(resolved, `Missing import target for ${path.relative(appRoot, file)} -> ${specifier}`);
    }
  }
}

async function collectFiles(root) {
  const entries = await readdir(root, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    const target = path.join(root, entry.name);
    if (entry.isDirectory()) {
      files.push(...await collectFiles(target));
      continue;
    }
    files.push(target);
  }
  return files;
}

async function assertPathExists(target, message) {
  try {
    await stat(target);
  } catch (_error) {
    throw new Error(message);
  }
}

async function assertModuleTarget(target, message) {
  const candidates = [target, `${target}.js`, path.join(target, "index.js")];
  for (const candidate of candidates) {
    try {
      await stat(candidate);
      return;
    } catch (_error) {
    }
  }
  throw new Error(message);
}

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

main().catch((error) => {
  console.error(error.message || error);
  process.exit(1);
});
