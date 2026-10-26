import path from "node:path"
import { fileURLToPath } from "node:url"
import { createServer } from "./bridge"

// dist/index.js sits one level below the package root that holds bin/, skills/, and agents/.
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..")

export default {
  id: "@archcore/opencode-plugin",
  server: createServer(root),
}
