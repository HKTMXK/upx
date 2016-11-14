package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"
	"time"
)

var (
	username string
	password string
	bucket   string
	tmpPath  string
	diskPath string
	fs       *fakeFs
)

type fakeFs struct {
	fileSet map[string]bool
	dirSet  map[string]bool
}

func (fs *fakeFs) Add(name string, isDir bool) {
	if !isDir {
		fs.fileSet[name] = true
		name = path.Dir(name)
	}
	for name != "/" {
		fs.dirSet[name] = true
		name = path.Dir(name)
	}
}

func (fs *fakeFs) Rm(name string) {
	for k, _ := range fs.fileSet {
		if strings.HasPrefix(k, name) {
			delete(fs.fileSet, k)
		}
	}
	for k, _ := range fs.dirSet {
		if strings.HasPrefix(k, name) {
			delete(fs.dirSet, k)
		}
	}
}

func (fs *fakeFs) List(name string) (files, dirs []string) {
	for k, _ := range fs.fileSet {
		if path.Dir(k) == name {
			files = append(files, path.Base(k))
		}
	}
	for k, _ := range fs.dirSet {
		if path.Dir(k) == name {
			dirs = append(dirs, path.Base(k))
		}
	}
	return
}

func (fs *fakeFs) Check(t *testing.T) {
	for v, _ := range fs.fileSet {
		b, err := upx("ls", v)
		check(t, err == nil && strings.Contains(string(b), path.Base(v)),
			"failed to ls %s %s %v", v, string(b), err)
	}
	for v, _ := range fs.dirSet {
		b, err := upx("ls", v)
		check(t, err == nil, "failed to ls %s %v", v, err)
		files, dirs := fs.List(v)
		all := len(files) + len(dirs)
		cnt := strings.Count(string(b), "\n")
		check(t, cnt == all, "%s %d %d not equal %v %v %v", v, cnt, all, files, dirs, string(b))

		for _, f := range files {
			check(t, strings.Contains(string(b), f), "not found %s %s", v, f)
		}
		for _, dir := range dirs {
			check(t, strings.Contains(string(b), dir), "not found %s %s", v, dir)
		}
	}
}

func init() {
	username = os.Getenv("username")
	password = os.Getenv("password")
	bucket = os.Getenv("bucket")
	tmpPath = fmt.Sprintf("/upx-%d", time.Now().Unix())
	diskPath = fmt.Sprintf("local-%d", time.Now().Unix())

	makeEnv(diskPath)

	if username == "" {
		fmt.Fprintf(os.Stderr, "username not set\n")
		os.Exit(-1)
	}
	if password == "" {
		fmt.Fprintf(os.Stderr, "password not set\n")
		os.Exit(-1)
	}
	if bucket == "" {
		fmt.Fprintf(os.Stderr, "bucket not set\n")
		os.Exit(-1)
	}

	path := os.Getenv("PATH")
	pwd, _ := os.Getwd()
	os.Setenv("PATH", path+":"+pwd)

	fs = &fakeFs{
		dirSet:  make(map[string]bool),
		fileSet: make(map[string]bool),
	}
}

func makeEnv(prefix string) {
	x := []string{
		"/foo/empty/",
		"/foo/.dir/dir2/file",
		"/foo/.dir/a/",
		"/foo/.dir2/b/1",
		"/foo/.dir2/b/2",
		"/foo/.dir3/c/3",
		"/foo/haha.go",
	}

	for k, s := range x {
		p := path.Join(prefix, s)
		if strings.HasSuffix(s, "/") {
			err := os.MkdirAll(p, 0700)
			checkExit(err == nil, "failed to mkdir", err)
		} else {
			err := os.MkdirAll(path.Dir(p), 0700)
			checkExit(err == nil, "failed to mkdir", err)

			fd, err := os.Create(p)
			checkExit(err == nil, "failed to create", err)
			if k%2 == 0 {
				_, err = fd.WriteString("")
			} else {
				_, err = fd.WriteString(string(p))
			}
			checkExit(err == nil, "failed to write", err)
			fd.Close()
		}
	}
}

func upx(comand string, args ...string) ([]byte, error) {
	args = append([]string{comand}, args...)
	cmd := exec.Command("./upx", args...)
	var obuf, ebuf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &obuf, &ebuf
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	err := cmd.Wait()
	ob, _ := ioutil.ReadAll(&obuf)
	eb, _ := ioutil.ReadAll(&ebuf)
	if err != nil {
		return ob, fmt.Errorf("%s", string(eb))
	}
	return ob, nil
}

func checkExit(cond bool, args ...interface{}) {
	if !cond {
		fmt.Fprintln(os.Stderr, args)
		os.Exit(-1)
	}
}

func check(t *testing.T, cond bool, arg0 string, args ...interface{}) {
	if !cond {
		if !strings.HasSuffix(arg0, "\n") {
			arg0 += "\n"
		}
		t.Errorf(arg0, args...)
		if t != nil {
			t.FailNow()
		}
	}
}

func makeDirName(n int) string {
	var letter = []rune("01234567890////abcdefg/////hijklmnopqrst.")
	b := make([]rune, n)
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}

func makeFileName(n int) string {
	var letter = []rune("01234567890abcdefghijklmnopqrst")
	b := make([]rune, n)
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}

func TestLogin(t *testing.T) {
	_, err := upx("login", bucket, username, password)
	check(t, err == nil, fmt.Sprintf("failed to upx %v", err))
}

func TestMkdir(t *testing.T) {
	_, err := upx("mkdir", tmpPath)
	check(t, err == nil, fmt.Sprintf("failed to upx %v", err))
	fs.Add(tmpPath, true)

	dir := tmpPath + "/../" + tmpPath + "/foo/../bar/"
	_, err = upx("mkdir", dir)
	check(t, err == nil, fmt.Sprintf("failed to upx %v", err))
	fs.Add(path.Join(dir), true)
}

func TestCd(t *testing.T) {
	_, err := upx("cd", tmpPath)
	check(t, err == nil, fmt.Sprintf("failed to upx %v", err))
}

func TestPwd(t *testing.T) {
	b, err := upx("pwd")
	check(t, err == nil, fmt.Sprintf("failed to upx %v", err))
	if string(b) != tmpPath+"\n" {
		t.Errorf("%s != %s\n", string(b), tmpPath)
		t.Fail()
	}
}

func TestPut(t *testing.T) {
	fname, desPath := "./upx.go", "../"+tmpPath+"/"
	_, err := upx("put", fname, desPath)
	check(t, err == nil, "failed to Put")
	fs.Add(path.Join(tmpPath, desPath, fname), false)

	desPath = fmt.Sprintf("empty-%d/", time.Now().Unix())
	_, err = upx("put", fname, desPath)
	check(t, err == nil, "failed to Put")
	fs.Add(path.Join(tmpPath, desPath, fname), false)

	desPath = fmt.Sprintf("newname")
	_, err = upx("put", fname, desPath)
	check(t, err == nil, "failed to Put")
	fs.Add(path.Join(tmpPath, desPath), false)

	_, err = upx("put", fname)
	check(t, err == nil, "failed to Put")
	fs.Add(path.Join(tmpPath, fname), false)

	_, err = upx("put", diskPath, "mustdir")

	var readDir func(name, prefix string)
	readDir = func(name, prefix string) {
		files, _ := ioutil.ReadDir(name)
		for _, f := range files {
			if f.IsDir() {
				fs.Add(path.Join(tmpPath, prefix, name, f.Name()), true)
				readDir(path.Join(name, f.Name()), prefix)
			} else {
				fs.Add(path.Join(tmpPath, prefix, name, f.Name()), false)
			}
		}
	}

	readDir(diskPath, "mustdir")

	fs.Check(t)
}

func TestLs(t *testing.T) {
	t.Skipf("ls is everywhere")
}

func TestGet(t *testing.T) {
	_, err := upx("get", "upx.go", "upx.go.2")
	check(t, err == nil, fmt.Sprintf("failed to upx %v", err))

	_, err = os.Lstat("upx.go.2")
	if err != nil {
		t.Errorf("failed to upx")
		t.FailNow()
	}
}

func TestSync(t *testing.T) {
	_, err := upx("sync", diskPath, path.Join(tmpPath, "sync", diskPath))
	check(t, err == nil, "failed to sync")

	var readDir func(name, prefix string)
	readDir = func(name, prefix string) {
		files, _ := ioutil.ReadDir(name)
		for _, f := range files {
			if f.IsDir() {
				fs.Add(path.Join(tmpPath, prefix, name, f.Name()), true)
				readDir(path.Join(name, f.Name()), prefix)
			} else {
				fs.Add(path.Join(tmpPath, prefix, name, f.Name()), false)
			}
		}
	}
	readDir(diskPath, "sync")
	fs.Check(t)

	fd, _ := os.Create("newer")
	fd.WriteString("xx")
	fd.Close()

	_, err = upx("sync", diskPath, path.Join(tmpPath, "sync", diskPath))
	check(t, err == nil, "failed to sync")

	readDir(diskPath, "sync")
	fs.Check(t)
}

func TestServices(t *testing.T) {
	b, err := upx("services")
	check(t, err == nil, fmt.Sprintf("failed to upx %v", err))
	if !strings.Contains(string(b), bucket) {
		t.FailNow()
	}

	b1, err := upx("sv")
	check(t, err == nil, fmt.Sprintf("failed to upx %v", err))
	if string(b) != string(b1) {
		t.Errorf("%s != %s\n", string(b), string(b1))
		t.Fail()
	}
}

func TestSwitch(t *testing.T) {
	_, err := upx("switch", bucket)
	check(t, err == nil, fmt.Sprintf("failed to upx %v", err))
}

func TestRmDir(t *testing.T) {
	for k := range fs.dirSet {
		if len(k) > len(tmpPath) {
			_, err := upx("rm", "-d", k)
			check(t, err == nil, "failed to rm")
			fs.Rm(k)
			break
		}
	}
	fs.Check(t)
}

func TestRmFile(t *testing.T) {
	for k := range fs.fileSet {
		if len(k) > len(tmpPath) {
			_, err := upx("rm", k)
			check(t, err == nil, "failed to rm")
			fs.Rm(k)
			break
		}
	}
	fs.Check(t)
}

func TestRmAll(t *testing.T) {
	_, err := upx("rm", "-a", tmpPath+"/*")
	check(t, err == nil, "failed to rm")
	fs.Rm(tmpPath + "/")

	_, err = upx("rm", "-a", tmpPath)
	fs.Rm(tmpPath)
	fs.Check(t)
}

func TestLogout(t *testing.T) {
	_, err := upx("logout")
	check(t, err == nil, fmt.Sprintf("failed to upx %v", err))
	os.RemoveAll(diskPath)
}
