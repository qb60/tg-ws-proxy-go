package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestManagerFWMarkLaunchAndAutostart(t *testing.T) {
	env := setEnvValue(managerEnv(t), "FW_MARK", "0x100")
	argsFile := filepath.Join(t.TempDir(), "args")
	env = append(env, "ARGS_FILE="+argsFile)
	writeCapturingProxyScript(t, envValue(env, "BIN_PATH"))
	if out, err := runManager(t, env, "start-background"); err != nil {
		t.Fatalf("start failed: %v\n%s", err, out)
	}
	t.Cleanup(func() { runManager(t, env, "stop") })
	waitForFile(t, argsFile)
	if args := readTrimmed(t, argsFile); !strings.Contains(args, "--fwmark\n0x100") {
		t.Fatalf("mark missing from manager arguments:\n%s", args)
	}
	if out, err := runManager(t, env, "enable-autostart"); err != nil {
		t.Fatalf("enable-autostart failed: %v\n%s", err, out)
	}
	if cfg := readTrimmed(t, envValue(env, "PERSIST_CONFIG_FILE")); !strings.Contains(cfg, "FW_MARK='0x100'") {
		t.Fatalf("mark missing from saved config:\n%s", cfg)
	}
	cmd := exec.Command("sh", "-c", `
        . "$1"
        procd_open_instance() { :; }
        procd_close_instance() { :; }
        procd_set_param() {
            if [ "$1" = command ]; then
                shift
                printf '%s\n' "$@"
            fi
        }
        start_service
    `, "_", envValue(env, "INIT_SCRIPT_PATH"))
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "--fwmark\n0x100") {
		t.Fatalf("procd arguments missing mark: %v\n%s", err, out)
	}
	if out, err := runManager(t, unsetEnvValue(env, "FW_MARK"), "telegram"); err != nil || !strings.Contains(out, "fwmark   : 0x100") {
		t.Fatalf("saved mark not loaded: %v\n%s", err, out)
	}
	if out, err := runManager(t, setEnvValue(env, "FW_MARK", "0"), "telegram"); err != nil || !strings.Contains(out, "fwmark   : 0 ") {
		t.Fatalf("environment did not override saved mark: %v\n%s", err, out)
	}
}

func TestManagerFWMarkMenu(t *testing.T) {
	env := setEnvValue(managerEnv(t), "FW_MARK", "0")
	out, err := runManagerMenu(t, env, "4\n22\n0x100\n\n\n\n")
	if err != nil {
		t.Fatalf("menu failed: %v\n%s", err, out)
	}
	configFile := envValue(env, "PERSIST_CONFIG_FILE")
	before := readTrimmed(t, configFile)
	if !strings.Contains(before, "FW_MARK='0x100'") {
		t.Fatalf("menu did not save mark:\n%s", before)
	}
	out, err = runManagerMenu(t, env, "4\n22\n4294967296\n\n\n\n")
	if err != nil || !strings.Contains(out, "Mark must be between") {
		t.Fatalf("menu did not reject invalid mark: %v\n%s", err, out)
	}
	if after := readTrimmed(t, configFile); after != before {
		t.Fatal("invalid mark changed the saved configuration")
	}
}
