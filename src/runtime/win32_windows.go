package runtime

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"syscall"
	"unsafe"

	"github.com/jandedobbeleer/oh-my-posh/src/log"
	"github.com/jandedobbeleer/oh-my-posh/src/regex"

	"golang.org/x/sys/windows"
)

// win32 specific code

// win32 dll load and function definitions
var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowThreadProcessID = user32.NewProc("GetWindowThreadProcessId")

	psapi              = syscall.NewLazyDLL("psapi.dll")
	getModuleBaseNameA = psapi.NewProc("GetModuleBaseNameA")

	iphlpapi     = syscall.NewLazyDLL("iphlpapi.dll")
	hGetIfTable2 = iphlpapi.NewProc("GetIfTable2")
)

// enumWindows call enumWindows from user32 and returns all active windows
// https://docs.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-enumwindows
func enumWindows(enumFunc, lparam uintptr) (err error) {
	r1, _, e1 := syscall.SyscallN(procEnumWindows.Addr(), enumFunc, lparam, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

// getWindowText returns the title and text of a window from a window handle
// https://docs.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-getwindowtextw
func getWindowText(hwnd syscall.Handle, str *uint16, maxCount int32) (length int32, err error) {
	r0, _, e1 := syscall.SyscallN(procGetWindowTextW.Addr(), uintptr(hwnd), uintptr(unsafe.Pointer(str)), uintptr(maxCount))
	length = int32(r0)
	if length == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func getWindowFileName(handle syscall.Handle) (string, error) {
	var pid int
	_, _, _ = procGetWindowThreadProcessID.Call(uintptr(handle), uintptr(unsafe.Pointer(&pid)))
	const query = windows.PROCESS_QUERY_INFORMATION | windows.PROCESS_VM_READ
	h, err := windows.OpenProcess(query, false, uint32(pid))
	if err != nil {
		return "", errors.New("unable to open window process")
	}
	buf := [1024]byte{}
	length, _, _ := getModuleBaseNameA.Call(uintptr(h), 0, uintptr(unsafe.Pointer(&buf)), 1024)
	filename := string(buf[:length])
	return strings.ToLower(filename), nil
}

// GetWindowTitle searches for a window attached to the pid
func queryWindowTitles(processName, windowTitleRegex string) (string, error) {
	var title string
	// callback for EnumWindows
	cb := syscall.NewCallback(func(handle syscall.Handle, _ uintptr) uintptr {
		fileName, err := getWindowFileName(handle)
		if err != nil {
			// ignore the error and continue enumeration
			return 1
		}
		if processName != fileName {
			// ignore the error and continue enumeration
			return 1
		}
		b := make([]uint16, 200)
		_, err = getWindowText(handle, &b[0], int32(len(b)))
		if err != nil {
			// ignore the error and continue enumeration
			return 1
		}
		title = syscall.UTF16ToString(b)
		if regex.MatchString(windowTitleRegex, title) {
			// will cause EnumWindows to return 0 (error)
			// but we don't want to enumerate all windows since we got what we want
			return 0
		}
		return 1 // continue enumeration
	})
	// Enumerates all top-level windows on the screen
	// The error is not checked because if EnumWindows is stopped bofere enumerating all windows
	// it returns 0(error occurred) instead of 1(success)
	// In our case, title will equal "" or the title of the window anyway
	err := enumWindows(cb, 0)
	if title == "" {
		var message string
		if err != nil {
			message = err.Error()
		}
		return "", errors.New("no matching window title found\n" + message)
	}
	return title, nil
}

var (
	advapi     = syscall.NewLazyDLL("advapi32.dll")
	procGetAce = advapi.NewProc("GetAce")
)

const (
	ACCESS_DENIED_ACE_TYPE = 1 //nolint: revive
)

type accessMask uint32

func (m accessMask) canWrite() bool {
	allowed := []int{windows.GENERIC_WRITE, windows.WRITE_DAC, windows.WRITE_OWNER}
	for _, v := range allowed {
		if m&accessMask(v) != 0 {
			return true
		}
	}
	return false
}

func (m accessMask) permissions() string {
	var permissions []string
	if m&windows.GENERIC_READ != 0 {
		permissions = append(permissions, "GENERIC_READ")
	}
	if m&windows.GENERIC_WRITE != 0 {
		permissions = append(permissions, "GENERIC_WRITE")
	}
	if m&windows.GENERIC_EXECUTE != 0 {
		permissions = append(permissions, "GENERIC_EXECUTE")
	}
	if m&windows.GENERIC_ALL != 0 {
		permissions = append(permissions, "GENERIC_ALL")
	}
	if m&windows.WRITE_DAC != 0 {
		permissions = append(permissions, "WRITE_DAC")
	}
	if m&windows.WRITE_OWNER != 0 {
		permissions = append(permissions, "WRITE_OWNER")
	}
	if m&windows.SYNCHRONIZE != 0 {
		permissions = append(permissions, "SYNCHRONIZE")
	}
	if m&windows.DELETE != 0 {
		permissions = append(permissions, "DELETE")
	}
	if m&windows.READ_CONTROL != 0 {
		permissions = append(permissions, "READ_CONTROL")
	}
	if m&windows.ACCESS_SYSTEM_SECURITY != 0 {
		permissions = append(permissions, "ACCESS_SYSTEM_SECURITY")
	}
	if m&windows.MAXIMUM_ALLOWED != 0 {
		permissions = append(permissions, "MAXIMUM_ALLOWED")
	}
	return strings.Join(permissions, "\n")
}

type AccessAllowedAce struct {
	AceType    uint8
	AceFlags   uint8
	AceSize    uint16
	AccessMask accessMask
	SidStart   uint32
}

func getCurrentUser() (user *tokenUser, err error) {
	token := windows.GetCurrentProcessToken()
	defer token.Close()

	tokenuser, err := token.GetTokenUser()
	if err != nil {
		return
	}
	tokenGroups, err := token.GetTokenGroups()
	if err != nil {
		return
	}
	user = &tokenUser{
		sid:    tokenuser.User.Sid,
		groups: tokenGroups.AllGroups(),
	}
	return
}

type tokenUser struct {
	sid    *windows.SID
	groups []windows.SIDAndAttributes
}

func (u *tokenUser) isMemberOf(sid *windows.SID) bool {
	if u.sid.Equals(sid) {
		return true
	}
	for _, g := range u.groups {
		if g.Sid.Equals(sid) {
			return true
		}
	}
	return false
}

func (env *Terminal) isWriteable(folder string) bool {
	cu, err := getCurrentUser()

	if err != nil {
		// unable to get current user
		log.Error(err)
		return false
	}

	si, err := windows.GetNamedSecurityInfo(folder, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		log.Error(err)
		return false
	}

	dacl, _, err := si.DACL()
	if err != nil || dacl == nil {
		// no dacl implies full access
		log.Debug("no dacl")
		return true
	}

	rs := reflect.ValueOf(dacl).Elem()
	aceCount := rs.Field(3).Uint()

	for i := range aceCount {
		ace := &AccessAllowedAce{}

		ret, _, _ := procGetAce.Call(uintptr(unsafe.Pointer(dacl)), uintptr(i), uintptr(unsafe.Pointer(&ace)))
		if ret == 0 {
			log.Debug("no ace found")
			return false
		}

		aceSid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))

		if !cu.isMemberOf(aceSid) {
			log.Debug("not current user or in group")
			continue
		}

		log.Debug(fmt.Sprintf("current user is member of %s", aceSid.String()))

		// this gets priority over the other access types
		if ace.AceType == ACCESS_DENIED_ACE_TYPE {
			log.Debug("ACCESS_DENIED_ACE_TYPE")
			return false
		}

		log.Debugf("%v", ace.AccessMask.permissions())
		if ace.AccessMask.canWrite() {
			log.Debug("user has write access")
			return true
		}
	}
	log.Debug("no write access")
	return false
}

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	globalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")
)

type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

func (env *Terminal) Memory() (*Memory, error) {
	var memStat memoryStatusEx
	memStat.Length = uint32(unsafe.Sizeof(memStat))
	r0, _, err := globalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&memStat)))
	if r0 == 0 {
		log.Error(err)
		return nil, err
	}
	return &Memory{
		PhysicalTotalMemory:     memStat.TotalPhys,
		PhysicalFreeMemory:      memStat.AvailPhys,
		PhysicalAvailableMemory: memStat.AvailPhys,
		PhysicalPercentUsed:     float64(memStat.MemoryLoad),
	}, nil
}
