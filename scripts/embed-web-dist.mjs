import { cpSync, mkdirSync, mkdtempSync, rmSync, renameSync } from "node:fs";
import { dirname, join, resolve } from "node:path";

function isNodeError(error, code) {
  return error instanceof Error && "code" in error && error.code === code;
}

function movePathSync(sourcePath, destinationPath) {
  try {
    renameSync(sourcePath, destinationPath);
  } catch (error) {
    if (!isNodeError(error, "EXDEV")) {
      throw error;
    }
    cpSync(sourcePath, destinationPath, { recursive: true, dereference: true });
    rmSync(sourcePath, { recursive: true, force: true });
  }
}

const source = process.argv[2];
const destination = process.argv[3];

if (!source || !destination) {
  throw new Error("usage: node scripts/embed-web-dist.mjs <source> <destination>");
}

const sourcePath = resolve(source);
const destinationPath = resolve(destination);
const destinationParent = dirname(destinationPath);
mkdirSync(destinationParent, { recursive: true });
const stagingRoot = mkdtempSync(join(destinationParent, ".dist-stage-"));
const stagedPath = join(stagingRoot, "dist");
const backupPath = join(destinationParent, `.dist-backup-${process.pid}-${Date.now()}`);
let movedExisting = false;

try {
  cpSync(sourcePath, stagedPath, { recursive: true, dereference: true });
  try {
    movePathSync(destinationPath, backupPath);
    movedExisting = true;
  } catch (error) {
    if (!isNodeError(error, "ENOENT")) {
      throw error;
    }
  }
  try {
    movePathSync(stagedPath, destinationPath);
  } catch (error) {
    if (movedExisting) {
      rmSync(destinationPath, { recursive: true, force: true });
      movePathSync(backupPath, destinationPath);
      movedExisting = false;
    }
    throw error;
  }
  if (movedExisting) {
    rmSync(backupPath, { recursive: true, force: true });
  }
} finally {
  rmSync(stagingRoot, { recursive: true, force: true });
}
