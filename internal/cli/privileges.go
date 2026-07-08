package cli

import (
	"fmt"
	"os"
)

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

func requireRoot(commandName string) error {
	if checkSuperuserPrivileges() != User {
		return nil
	}

	return fmt.Errorf("Portico needs root privileges to %s. Please run with sudo or as root", commandName)
}
