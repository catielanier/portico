package cli

import "os"

type UserPrivilegeLevel string

const (
	Superuser UserPrivilegeLevel = "superuser"
	Sudo      UserPrivilegeLevel = "sudo"
	User      UserPrivilegeLevel = "user"
)

func checkSuperuserPrivileges() UserPrivilegeLevel {
	isRoot := os.Geteuid() == 0

	sudoUser := os.Getenv("SUDO_USER")
	isSudo := isRoot && sudoUser != ""

	if isRoot && isSudo {
		return Sudo
	} else if isRoot {
		return Superuser
	} else {
		return User
	}
}
