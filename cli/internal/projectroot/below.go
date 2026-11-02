package projectroot

import (
	"os"
	"path/filepath"
	"strings"
)

// Below lists the project roots under dir, breadth first and in name order, so
// one tree always yields one list. It descends at most maxDepth levels and
// reads at most maxDirs directories; complete is false when either bound cut
// the walk. A project's own subtree is not entered: a nested project is found
// by the walk upward from a file, not by this inventory.
//
// Hidden directories, dependency trees, and plugin install caches are skipped:
// none of them is a project a user opens.
//
// An advisory read: a directory that cannot be read is skipped.
func Below(dir string, maxDepth, maxDirs int) (roots []string, complete bool) {
	type entry struct {
		path  string
		depth int
	}
	queue := []entry{{path: filepath.Clean(dir)}}
	read := 0
	complete = true
	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		if read >= maxDirs {
			return roots, false
		}
		read++
		entries, err := os.ReadDir(next.path)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || skipBelow(e.Name()) {
				continue
			}
			child := filepath.Join(next.path, e.Name())
			if IsPluginCachePath(child) {
				continue
			}
			if info, err := os.Stat(filepath.Join(child, ".archcore")); err == nil && info.IsDir() {
				roots = append(roots, child)
				continue
			}
			if next.depth+1 < maxDepth {
				queue = append(queue, entry{path: child, depth: next.depth + 1})
			} else {
				complete = false
			}
		}
	}
	return roots, complete
}

func skipBelow(name string) bool {
	return strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor"
}
