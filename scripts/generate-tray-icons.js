#!/usr/bin/env node

/**
 * Generate tray assets under resources/tray from a source PNG (default: public/logo.png).
 *
 * 1) Prefer ImageMagick (`magick` or `convert`) for full parity (multi-size .ico, macOS template).
 * 2) If ImageMagick is missing, use sharp + png-to-ico (no system install; run `npm install`).
 */

const fs = require('fs');
const os = require('os');
const path = require('path');
const { spawnSync } = require('child_process');

const projectRoot = path.resolve(__dirname, '..');
const inputPath = path.resolve(projectRoot, process.argv[2] || 'build/icons/png/512x512.png');
const outputDir = path.resolve(projectRoot, 'resources/tray');

function run(cmd, args) {
  const result = spawnSync(cmd, args, { stdio: 'pipe', encoding: 'utf8' });
  if (result.status !== 0) {
    const stderr = result.stderr?.trim();
    const stdout = result.stdout?.trim();
    const detail = stderr || stdout || `exit code ${result.status}`;
    throw new Error(`${cmd} ${args.join(' ')} failed: ${detail}`);
  }
}

function hasCommand(cmd, args) {
  const result = spawnSync(cmd, args, { stdio: 'ignore' });
  return result.status === 0;
}

function ensureImageMagick() {
  if (hasCommand('magick', ['-version'])) return 'magick';
  if (hasCommand('convert', ['-version'])) return 'convert';
  return null;
}

function ensureInputExists() {
  if (!fs.existsSync(inputPath)) {
    throw new Error(`Input logo not found: ${inputPath}`);
  }
}

function ensureOutputDir() {
  fs.mkdirSync(outputDir, { recursive: true });
}

function mainImageMagick(magick) {
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'tray-icons-'));

  const win16 = path.join(tmpDir, 'tray-16.png');
  const win32 = path.join(tmpDir, 'tray-32.png');
  const win48 = path.join(tmpDir, 'tray-48.png');

  const linuxPng = path.join(outputDir, 'tray-icon.png');
  const winIco = path.join(outputDir, 'tray-icon.ico');
  const macTemplate = path.join(outputDir, 'trayIconTemplate.png');
  const macTemplate2x = path.join(outputDir, 'trayIconTemplate@2x.png');
  const macColor = path.join(outputDir, 'tray-icon-mac.png');
  const macColor2x = path.join(outputDir, 'tray-icon-mac@2x.png');
  const macColorRaw = path.join(tmpDir, 'tray-icon-mac-raw.png');
  const macColor2xRaw = path.join(tmpDir, 'tray-icon-mac-2x-raw.png');

  run(magick, [inputPath, '-resize', '48x48', linuxPng]);

  run(magick, [inputPath, '-resize', '16x16', win16]);
  run(magick, [inputPath, '-resize', '32x32', win32]);
  run(magick, [inputPath, '-resize', '48x48', win48]);
  run(magick, [win16, win32, win48, winIco]);

  run(magick, [
    inputPath, '-resize', '18x18',
    '-colorspace', 'Gray', '-threshold', '70%',
    '-alpha', 'copy',
    '-channel', 'RGB', '-fill', 'black', '-colorize', '100',
    '-trim', '+repage',
    '-background', 'none', '-gravity', 'center', '-extent', '18x18',
    macTemplate,
  ]);

  run(magick, [
    inputPath, '-resize', '36x36',
    '-colorspace', 'Gray', '-threshold', '70%',
    '-alpha', 'copy',
    '-channel', 'RGB', '-fill', 'black', '-colorize', '100',
    '-trim', '+repage',
    '-background', 'none', '-gravity', 'center', '-extent', '36x36',
    macTemplate2x,
  ]);

  run(magick, [
    inputPath,
    '-trim', '+repage',
    '-resize', '16x16',
    '-modulate', '108,118,100',
    '-sigmoidal-contrast', '4,50%',
    '-background', 'none', '-gravity', 'center', '-extent', '18x18',
    macColorRaw,
  ]);

  run(magick, [
    inputPath,
    '-trim', '+repage',
    '-resize', '32x32',
    '-modulate', '108,118,100',
    '-sigmoidal-contrast', '4,50%',
    '-background', 'none', '-gravity', 'center', '-extent', '36x36',
    macColor2xRaw,
  ]);

  run(magick, [
    macColorRaw,
    '-alpha', 'on',
    '-colorspace', 'sRGB',
    '-type', 'TrueColorAlpha',
    '-strip',
    '-define', 'png:color-type=6',
    macColor,
  ]);

  run(magick, [
    macColor2xRaw,
    '-alpha', 'on',
    '-colorspace', 'sRGB',
    '-type', 'TrueColorAlpha',
    '-strip',
    '-define', 'png:color-type=6',
    macColor2x,
  ]);

  fs.rmSync(tmpDir, { recursive: true, force: true });
  console.log(`Generated tray icons (ImageMagick) from ${inputPath} -> ${outputDir}`);
}

async function mainSharpFallback() {
  let sharp;
  let pngToIco;
  try {
    sharp = require('sharp');
    pngToIco = require('png-to-ico');
  } catch {
    throw new Error(
      'ImageMagick not found and Node fallback deps missing. Either install ImageMagick (https://imagemagick.org) ' +
        'so `magick` or `convert` is on PATH, or run `npm install` in the repo (adds sharp + png-to-ico), ' +
        'then retry. If npm install fails with EBUSY, quit Electron and other apps using this folder first.'
    );
  }

  const transparent = { r: 0, g: 0, b: 0, alpha: 0 };
  const resizeOpts = { fit: 'contain', position: 'center', background: transparent };

  const buf16 = await sharp(inputPath).resize(16, 16, resizeOpts).png().toBuffer();
  const buf32 = await sharp(inputPath).resize(32, 32, resizeOpts).png().toBuffer();
  const buf48 = await sharp(inputPath).resize(48, 48, resizeOpts).png().toBuffer();

  fs.writeFileSync(path.join(outputDir, 'tray-icon.png'), buf48);

  const icoBuf = await pngToIco([buf16, buf32, buf48]);
  fs.writeFileSync(path.join(outputDir, 'tray-icon.ico'), icoBuf);

  const mac = await sharp(inputPath).resize(18, 18, resizeOpts).png().toBuffer();
  fs.writeFileSync(path.join(outputDir, 'tray-icon-mac.png'), mac);
  const mac2x = await sharp(inputPath).resize(36, 36, resizeOpts).png().toBuffer();
  fs.writeFileSync(path.join(outputDir, 'tray-icon-mac@2x.png'), mac2x);

  console.log(`Generated tray icons (sharp fallback) from ${inputPath} -> ${outputDir}`);
  console.warn(
    'Note: trayIconTemplate.png / @2x were not generated without ImageMagick; macOS menu bar template styling may differ.'
  );
}

async function main() {
  ensureInputExists();
  ensureOutputDir();

  const magick = ensureImageMagick();
  if (magick) {
    mainImageMagick(magick);
    return;
  }

  await mainSharpFallback();
}

(async () => {
  try {
    await main();
  } catch (error) {
    console.error(error instanceof Error ? error.message : String(error));
    process.exit(1);
  }
})();
