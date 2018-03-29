package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang/dep"
)

func hasSymlink(path string) bool {
	sympath, _ := filepath.EvalSymlinks(path)
	return sympath != path
}

func realMain() int {
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "fatal: Must specify a target project to link.")
		return 1
	} else if flag.NArg() > 1 {
		fmt.Fprintln(os.Stderr, "fatal: Must specify exactly one argument.")
		return 1
	}
	target := flag.Arg(0)

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: Unable to detect current working directory: %s.\n", err)
		return 1
	}

	ctx := &dep.Ctx{
		Out:            log.New(os.Stdout, "", 0),
		Err:            log.New(os.Stderr, "", 0),
		DisableLocking: os.Getenv("DEPNOLOCK") != "",
	}

	GOPATHS := filepath.SplitList(os.Getenv("GOPATH"))
	ctx.SetPaths(wd, GOPATHS...)

	project, err := ctx.LoadProject()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: Could not load project: %s.\n", err)
		return 1
	}

	vendorDir := filepath.Join(project.AbsRoot, "vendor", target)
	// Verify that the directory for the vendor directory does not evaluate to have any symlinks.
	if hasSymlink(filepath.Dir(vendorDir)) {
		fmt.Fprintln(os.Stderr, "fatal: Found a symlink in the vendor directory path.")
		return 1
	}

	// Check to see if the vendor directory itself is a symlink.
	st, err := os.Lstat(vendorDir)
	if os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "fatal: Target directory does not exist.")
		return 1
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: Unable to stat the vendor directory: %s\n", err)
		return 1
	}

	// Detect the proper symlink target. If we have a symlink, ensure the symlink points to that path.
	// If it isn't a symlink, remove the directory and create the symlink.
	targetDir, err := ctx.AbsForImport(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: Could not detect import path for the target: %s\n", err)
		return 1
	}

	if st.Mode()&os.ModeSymlink != 0 {
		symlink, err := os.Readlink(vendorDir)
		if err == nil && symlink == targetDir {
			return 0
		}
		// Remove the link and recreate it.
		os.Remove(vendorDir)
	} else {
		// Remove the target directory.
		os.RemoveAll(vendorDir)
	}

	// Create the symlink.
	os.Symlink(targetDir, vendorDir)
	return 0
}

func main() {
	os.Exit(realMain())
}
