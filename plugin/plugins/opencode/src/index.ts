import path from "node:path"
import { fileURLToPath } from "node:url"
import { createServer } from "./bridge"
import { createSetup } from "./v2"

// dist/index.js sits one level below the package root that holds bin/, skills/, and agents/.
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..")

// OpenCode 1.x calls server and OpenCode 2.x calls setup; each half drops the
// other key (opencode-plugin-serves-v1-and-v2-from-one-package.adr).
export default {
  id: "@archcore/opencode-plugin",
  server: createServer(root),
  setup: createSetup(root),
}
