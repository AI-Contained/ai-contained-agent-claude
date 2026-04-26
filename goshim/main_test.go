package main

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type fileSpec struct {
	name     string
	contents string
	perm     fs.FileMode
}

// writeFile materializes a file under root and returns its spec for direct
// equality assertions. Permissions are applied explicitly to bypass umask.
func writeFile(root, name, contents string, perm fs.FileMode) fileSpec {
	full := filepath.Join(root, name)
	Expect(os.MkdirAll(filepath.Dir(full), 0755)).To(Succeed())
	Expect(os.WriteFile(full, []byte(contents), perm)).To(Succeed())
	Expect(os.Chmod(full, perm)).To(Succeed())
	return fileSpec{name: name, contents: contents, perm: perm}
}

// readFile loads the file at root/name into a fileSpec for direct equality assertions.
func readFile(root, name string) fileSpec {
	full := filepath.Join(root, name)
	data, err := os.ReadFile(full)
	Expect(err).NotTo(HaveOccurred())
	info, err := os.Stat(full)
	Expect(err).NotTo(HaveOccurred())
	return fileSpec{
		name:     name,
		contents: string(data),
		perm:     info.Mode().Perm(),
	}
}

var _ = Describe("mirror", func() {
	var (
		src string
		dst string
		log *bytes.Buffer
	)

	BeforeEach(func() {
		src = GinkgoT().TempDir()
		dst = GinkgoT().TempDir()
		log = &bytes.Buffer{}
	})

	DescribeTable("copies missing entries from src to dst",
		func(name, contents string, perm fs.FileMode) {
			expected := writeFile(src, name, contents, perm)
			Expect(mirror(src, dst, log)).To(Succeed())
			Expect(readFile(dst, expected.name)).To(Equal(expected))
		},
		Entry("regular file",     "settings.json",    `{"a":1}`, fs.FileMode(0644)),
		Entry("dotfile",          ".claude.json",     `{}`,      fs.FileMode(0644)),
		Entry("nested file",      "sub/nested.json",  `{}`,      fs.FileMode(0644)),
		Entry("restricted perms", "secret.json",      `{}`,      fs.FileMode(0600)),
	)

	It("logs each created path", func() {
		expected := writeFile(src, "settings.json", `{}`, 0644)
		Expect(mirror(src, dst, log)).To(Succeed())
		Expect(log.String()).To(ContainSubstring(expected.name))
	})

	It("copies multiple missing files in a single pass", func() {
		a := writeFile(src, "a.json", `{"a":1}`, 0644)
		b := writeFile(src, "b.json", `{"b":2}`, 0644)
		Expect(mirror(src, dst, log)).To(Succeed())
		Expect(readFile(dst, a.name)).To(Equal(a))
		Expect(readFile(dst, b.name)).To(Equal(b))
	})

	It("copies only the missing entries when dst is partially populated", func() {
		missing := writeFile(src, "new.json", `{"new":true}`, 0644)
		writeFile(src, "existing.json", `{"src":true}`, 0644)
		existing := writeFile(dst, "existing.json", `{"dst":true}`, 0600)
		Expect(mirror(src, dst, log)).To(Succeed())
		Expect(readFile(dst, missing.name)).To(Equal(missing))
		Expect(readFile(dst, existing.name)).To(Equal(existing))
	})

	Context("when dst already has matching entries", func() {
		It("leaves existing files untouched and unlogged", func() {
			writeFile(src, "settings.json", `{"new":true}`, 0644)
			expected := writeFile(dst, "settings.json", `{"old":true}`, 0644)
			Expect(mirror(src, dst, log)).To(Succeed())
			Expect(readFile(dst, expected.name)).To(Equal(expected))
			Expect(log.String()).NotTo(ContainSubstring(expected.name))
		})

		It("copies new files into an existing subdirectory", func() {
			expected := writeFile(src, "sub/nested.json", `{}`, 0644)
			Expect(os.Mkdir(filepath.Join(dst, "sub"), 0755)).To(Succeed())
			Expect(mirror(src, dst, log)).To(Succeed())
			Expect(readFile(dst, expected.name)).To(Equal(expected))
		})
	})

	It("preserves directory permissions", func() {
		Expect(os.Mkdir(filepath.Join(src, "sub"), 0700)).To(Succeed())
		Expect(os.Chmod(filepath.Join(src, "sub"), 0700)).To(Succeed())
		Expect(mirror(src, dst, log)).To(Succeed())

		info, err := os.Stat(filepath.Join(dst, "sub"))
		Expect(err).NotTo(HaveOccurred())
		Expect(info.Mode().Perm()).To(Equal(fs.FileMode(0700)))
	})

	It("creates empty directories from src", func() {
		Expect(os.Mkdir(filepath.Join(src, "logs"), 0755)).To(Succeed())
		Expect(mirror(src, dst, log)).To(Succeed())

		info, err := os.Stat(filepath.Join(dst, "logs"))
		Expect(err).NotTo(HaveOccurred())
		Expect(info.IsDir()).To(BeTrue())
	})

	It("returns an error when an existing dst subdirectory is not writable", func() {
		if os.Geteuid() == 0 {
			Skip("running as root bypasses permission checks")
		}
		writeFile(src, "sub/nested.json", `{}`, 0644)
		Expect(os.Mkdir(filepath.Join(dst, "sub"), 0500)).To(Succeed())
		Expect(os.Chmod(filepath.Join(dst, "sub"), 0500)).To(Succeed())

		Expect(mirror(src, dst, log)).To(MatchError(fs.ErrPermission))
	})
})
