// Updates the Flatpak manifest with the latest values for the staged artifact
// Usage: node generate-flatpak-manifest.js <path-to-staging-directory>

const path = require("path");
const fs = require("fs");
const yaml = require("yaml");

async function generateFlatpakManifest(stagingDir) {
    const latestYmlPath = path.join(stagingDir, "latest-linux.yml");
    const latestYml = yaml.parse(fs.readFileSync(latestYmlPath, "utf8"));
    const manifestPath = path.join(stagingDir, "dev.commandline.waveterm.yml");
    const manifestYml = yaml.parse(fs.readFileSync(manifestPath, "utf8"));
    const latestDeb = latestYml.files.find((file) => file.url.endsWith(".deb"));
    const latestDebUrl = `https://dl.waveterm.dev/releases/${latestDeb.url}`;
    const latestDebSha512 = latestDeb.sha512;
    