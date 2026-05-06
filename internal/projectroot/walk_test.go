package projectroot

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWalk_FindsArchcoreFirst(t *testing.T) {
	t.Parallel()
	// Both .archcore and .git in the same dir — .archcore wins.
	root := mkTree(t, ".archcore/", ".git/")
	res, ok := walkUp(root, Options{
		Home: neutralHome(t),
		Stat: os.Stat,
	})
	if !ok {
		t.Fatalf("walkUp returned !ok for %s", root)
	}
	if res.Source != SourceWalkArchcore {
		t.Errorf("Source = %q, want %q", res.Source, SourceWalkArchcore)
	}
	if res.Marker != ".archcore" {
		t.Errorf("Marker = %q, want .archcore", res.Marker)
	}
}

func TestWalk_GitOverGoMod(t *testing.T) {
	t.Parallel()
	// Same level: .git wins over generic markers.
	root := mkTree(t, ".git/", "go.mod")
	res, ok := walkUp(root, Options{
		Home: neutralHome(t),
		Stat: os.Stat,
	})
	if !ok {
		t.Fatal("walkUp returned !ok")
	}
	if res.Source != SourceWalkGit {
		t.Errorf("Source = %q, want %q", res.Source, SourceWalkGit)
	}
}

func TestWalk_DepthCounter(t *testing.T) {
	t.Parallel()
	root := mkTree(t, ".archcore/", "a/b/c/")
	cwd := filepath.Join(root, "a", "b", "c")
	res, ok := walkUp(cwd, Options{
		Home: neutralHome(t),
		Stat: os.Stat,
	})
	if !ok {
		t.Fatal("walkUp returned !ok")
	}
	if res.WalkDepth != 3 {
		t.Errorf("WalkDepth = %d, want 3", res.WalkDepth)
	}
}

func TestWalk_NoMarkers(t *testing.T) {
	t.Parallel()
	// Empty tree: walk-up exhausts.
	cwd := mkTree(t, "a/b/c/")
	deep := filepath.Join(cwd, "a", "b", "c")
	_, ok := walkUp(deep, Options{
		Home: neutralHome(t),
		Stat: os.Stat,
	})
	if ok {
		t.Errorf("walkUp returned ok for empty tree")
	}
}

func TestWalk_HomeStillEvaluatedForMarkers(t *testing.T) {
	t.Parallel()
	// HOME has .git — walk must still evaluate markers at HOME and report
	// the hit. The HOME boundary applies *after* the marker check, so the
	// caller (Resolve+validate) gets a chance to convert it to ERR_HOME_REFUSED.
	home := mkTree(t, ".git/", "child/sub/")
	cwd := filepath.Join(home, "child", "sub")

	res, ok := walkUp(cwd, Options{
		Home: homeFn(home),
		Stat: os.Stat,
	})
	if !ok {
		t.Fatalf("walkUp returned !ok despite .git at HOME")
	}
	if res.Path != filepath.Clean(home) {
		t.Errorf("Path = %q, want %q", res.Path, home)
	}
	if res.Source != SourceWalkGit {
		t.Errorf("Source = %q, want %q", res.Source, SourceWalkGit)
	}
}

func TestWalk_ArchcoreClosestWins(t *testing.T) {
	t.Parallel()
	// /root/.archcore + /root/sub/.archcore + cwd at /root/sub/sub2.
	// Closest .archcore (/root/sub) should win.
	root := mkTree(t, ".archcore/", "sub/.archcore/", "sub/sub2/")
	cwd := filepath.Join(root, "sub", "sub2")
	res, ok := walkUp(cwd, Options{
		Home: neutralHome(t),
		Stat: os.Stat,
	})
	if !ok {
		t.Fatal("walkUp returned !ok")
	}
	want := filepath.Join(root, "sub")
	if res.Path != want {
		t.Errorf("Path = %q, want %q (closest .archcore)", res.Path, want)
	}
}

func TestWalk_GitClosestOverArchcoreFurther(t *testing.T) {
	t.Parallel()
	// /root/.archcore + /root/sub/.git, cwd at /root/sub/sub2.
	// .archcore at root takes priority (closest .archcore wins over closer .git).
	root := mkTree(t, ".archcore/", "sub/.git/", "sub/sub2/")
	cwd := filepath.Join(root, "sub", "sub2")
	res, ok := walkUp(cwd, Options{
		Home: neutralHome(t),
		Stat: os.Stat,
	})
	if !ok {
		t.Fatal("walkUp returned !ok")
	}
	if res.Source != SourceWalkArchcore {
		t.Errorf("Source = %q, want %q (.archcore wins globally)", res.Source, SourceWalkArchcore)
	}
	if res.Path != filepath.Clean(root) {
		t.Errorf("Path = %q, want %q", res.Path, root)
	}
}

func TestWalk_StopsAtFsRoot(t *testing.T) {
	t.Parallel()
	// Use an injected Stat that always reports "not found" so no marker can
	// trigger a hit anywhere in the walk path. Home is neutralized so the
	// only boundary is the filesystem root. The walk must terminate without
	// panic and return ok=false.
	notFoundStat := func(string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	}
	_, ok := walkUp(filepath.Clean(os.TempDir()), Options{
		Home: neutralHome(t),
		Stat: notFoundStat,
	})
	if ok {
		t.Errorf("walkUp returned ok despite Stat reporting no markers")
	}
}
