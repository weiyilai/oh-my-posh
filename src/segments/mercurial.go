package segments

import (
	"strings"

	"github.com/jandedobbeleer/oh-my-posh/src/runtime"
	"github.com/jandedobbeleer/oh-my-posh/src/runtime/path"
)

const (
	MERCURIALCOMMAND = "hg"

	hgLogTemplate = "{rev}|{node}|{branch}|{tags}|{bookmarks}"
)

type MercurialStatus struct {
	ScmStatus
}

func (s *MercurialStatus) add(code string) {
	switch code {
	case "R", "!":
		s.Deleted++
	case "A":
		s.Added++
	case "?":
		s.Untracked++
	case "M":
		s.Modified++
	}
}

type Mercurial struct {
	Working           *MercurialStatus
	LocalCommitNumber string
	ChangeSetID       string
	ChangeSetIDShort  string
	Branch            string
	scm
	Bookmarks []string
	Tags      []string
	IsTip     bool
}

func (hg *Mercurial) Template() string {
	return "hg {{.Branch}} {{if .LocalCommitNumber}}({{.LocalCommitNumber}}:{{.ChangeSetIDShort}}){{end}}{{range .Bookmarks }} \uf02e {{.}}{{end}}{{range .Tags}} \uf02b {{.}}{{end}}{{if .Working.Changed}} \uf044 {{ .Working.String }}{{ end }}" //nolint: lll
}

func (hg *Mercurial) Enabled() bool {
	if !hg.shouldDisplay() {
		return false
	}

	statusFormats := hg.props.GetKeyValueMap(StatusFormats, map[string]string{})
	hg.Working = &MercurialStatus{ScmStatus: ScmStatus{Formats: statusFormats}}

	displayStatus := hg.props.GetBool(FetchStatus, false)
	if displayStatus {
		hg.setMercurialStatus()
	}

	return true
}

func (hg *Mercurial) CacheKey() (string, bool) {
	dir, err := hg.env.HasParentFilePath(".hg", true)
	if err != nil {
		return "", false
	}

	return dir.Path, true
}

func (hg *Mercurial) shouldDisplay() bool {
	if !hg.hasCommand(MERCURIALCOMMAND) {
		return false
	}

	hgdir, err := hg.env.HasParentFilePath(".hg", false)
	if err != nil {
		return false
	}

	hg.setDir(hgdir.ParentFolder)

	hg.mainSCMDir = hgdir.Path
	hg.scmDir = hgdir.Path
	// convert the worktree file path to a windows one when in a WSL shared folder
	hg.repoRootDir = strings.TrimSuffix(hg.convertToWindowsPath(hgdir.Path), "/.hg")
	return true
}

func (hg *Mercurial) setDir(dir string) {
	dir = path.ReplaceHomeDirPrefixWithTilde(dir) // align with template PWD
	if hg.env.GOOS() == runtime.WINDOWS {
		hg.Dir = strings.TrimSuffix(dir, `\.hg`)
		return
	}
	hg.Dir = strings.TrimSuffix(dir, "/.hg")
}

func (hg *Mercurial) setMercurialStatus() {
	hg.Branch = hg.command

	idString := hg.getHgCommandOutput("log", "-r", ".", "--template", hgLogTemplate)
	if idString == "" {
		return
	}

	idSplit := strings.Split(idString, "|")
	if len(idSplit) != 5 {
		return
	}

	hg.LocalCommitNumber = idSplit[0]
	hg.ChangeSetID = idSplit[1]

	if len(hg.ChangeSetID) >= 12 {
		hg.ChangeSetIDShort = hg.ChangeSetID[:12]
	}
	hg.Branch = idSplit[2]

	hg.Tags = doSplit(idSplit[3])
	hg.Bookmarks = doSplit(idSplit[4])

	hg.IsTip = false
	tipIndex := 0
	for i, tag := range hg.Tags {
		if tag == "tip" {
			hg.IsTip = true
			tipIndex = i
			break
		}
	}

	if hg.IsTip {
		hg.Tags = RemoveAtIndex(hg.Tags, tipIndex)
	}

	statusString := hg.getHgCommandOutput("status")

	if statusString == "" {
		return
	}

	statusLines := strings.SplitSeq(statusString, "\n")

	for status := range statusLines {
		hg.Working.add(status[:1])
	}
}

func doSplit(s string) []string {
	if s == "" {
		return []string{}
	}

	return strings.Split(s, " ")
}

func RemoveAtIndex(s []string, index int) []string {
	ret := make([]string, 0)
	ret = append(ret, s[:index]...)
	return append(ret, s[index+1:]...)
}

func (hg *Mercurial) getHgCommandOutput(command string, args ...string) string {
	args = append([]string{"-R", hg.repoRootDir, command}, args...)
	val, err := hg.env.RunCommand(hg.command, args...)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(val)
}
