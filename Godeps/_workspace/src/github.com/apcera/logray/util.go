// Copyright 2012-2014 Apcera Inc. All rights reserved.

package logray

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// packageFilenameLine updates the LineData to include the package, function,
// source file, and line number.
func packageFilenameLine(ld *LineData, depth int) {
	i, filename, linenum, ok := runtime.Caller(depth)
	if !ok {
		return
	}

	// strip the directory from the filename
	fileParts := strings.Split(filename, string(os.PathSeparator))
	ld.SourceFile = fileParts[len(fileParts)-1]
	ld.SourceLine = linenum

	// Set up params from the call stack.
	f := runtime.FuncForPC(i)
	if f == nil {
		return
	}

	// generate the separate package name and function name
	packagePath := strings.Split(f.Name(), "/")
	n := len(packagePath)
	pkgFunc := strings.SplitN(packagePath[n-1], ".", 2)
	if len(pkgFunc) != 2 {
		return
	}

	pkg := strings.Join(packagePath[:n-1], "/") + fmt.Sprintf("/%s", pkgFunc[0])
	ld.CallingPackage = pkg
	ld.CallingFunction = pkgFunc[1]
}

// gatherStack generates a stack trace to attach to error messages.
func gatherStack() string {
	root := runtime.GOROOT()
	stack := make([]string, 0, 10)
	pc := make([]uintptr, 10)
	depth := runtime.Callers(4, pc)
	for i := 0; i < depth; i++ {
		f := runtime.FuncForPC(pc[i])
		file, line := f.FileLine(pc[i])
		if strings.HasPrefix(file, root) {
			continue
		}

		path := strings.Split(f.Name(), "/")
		pl := len(path)
		fun := path[pl-1]
		fa := strings.Split(fun, ".")
		if fa[0] == "server" && pl >= 2 {
			fa[0] = path[pl-2]
		}
		w := strings.Join(fa, ".")
		stack = append(stack, fmt.Sprintf("\t%s:%d %s", file, line, w))
	}
	return "\n" + strings.Join(stack, "\n")
}
