package portage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const SystemPortageConfigPath = "/etc/portage"

type ConfigSandbox struct {
	Root              string
	PortageConfigPath string
}

func NewConfigSandbox() (*ConfigSandbox, error) {
	root, err := os.MkdirTemp("", "portico-*")
	if err != nil {
		return nil, err
	}

	sandbox := &ConfigSandbox{
		Root:              root,
		PortageConfigPath: filepath.Join(root, "etc", "portage"),
	}

	if err := copyDir(SystemPortageConfigPath, sandbox.PortageConfigPath); err != nil {
		_ = os.RemoveAll(root)
		return nil, err
	}

	if err := fixSandboxMakeProfile(sandbox.PortageConfigPath); err != nil {
		_ = os.RemoveAll(root)
		return nil, err
	}

	return sandbox, nil
}

func (s *ConfigSandbox) Cleanup() error {
	if s == nil || s.Root == "" {
		return nil
	}

	return os.RemoveAll(s.Root)
}

func fixSandboxMakeProfile(sandboxConfigPath string) error {
	sourceProfile := filepath.Join(SystemPortageConfigPath, "make.profile")
	sandboxProfile := filepath.Join(sandboxConfigPath, "make.profile")

	resolvedProfile, err := filepath.EvalSymlinks(sourceProfile)
	if err != nil {
		return fmt.Errorf("failed to resolve %s: %w", sourceProfile, err)
	}

	if err := os.RemoveAll(sandboxProfile); err != nil {
		return err
	}

	if err := os.Symlink(resolvedProfile, sandboxProfile); err != nil {
		return fmt.Errorf("failed to create sandbox make.profile symlink: %w", err)
	}

	return nil
}

func copyDir(src string, dst string) error {
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return err
	}

	if srcInfo.Mode()&os.ModeSymlink != 0 {
		return copySymlink(src, dst)
	}

	if !srcInfo.IsDir() {
		return fmt.Errorf("%s is not a directory", src)
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		info, err := os.Lstat(srcPath)
		if err != nil {
			return err
		}

		if info.Mode()&os.ModeSymlink != 0 {
			if err := copySymlink(srcPath, dstPath); err != nil {
				return err
			}

			continue
		}

		if info.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}

			continue
		}

		if err := copyFile(srcPath, dstPath, info.Mode()); err != nil {
			return err
		}
	}

	return nil
}

func copyFile(src string, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Close()
}

func copySymlink(src string, dst string) error {
	target, err := os.Readlink(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	if err := os.RemoveAll(dst); err != nil {
		return err
	}

	return os.Symlink(target, dst)
}
