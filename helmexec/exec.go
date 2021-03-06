package helmexec

import (
	"io"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	command = "helm"
)

type execer struct {
	helmBinary  string
	runner      Runner
	logger      *zap.SugaredLogger
	kubeContext string
	extra       []string
}

func NewLogger(writer io.Writer, logLevel string) *zap.SugaredLogger {
	var cfg zapcore.EncoderConfig
	cfg.MessageKey = "message"
	out := zapcore.AddSync(writer)
	var level zapcore.Level
	err := level.Set(logLevel)
	if err != nil {
		panic(err)
	}
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(cfg),
		out,
		level,
	)
	return zap.New(core).Sugar()
}

// New for running helm commands
func New(logger *zap.SugaredLogger, kubeContext string) *execer {
	return &execer{
		helmBinary:  command,
		logger:      logger,
		kubeContext: kubeContext,
		runner:      &ShellRunner{},
	}
}

func (helm *execer) SetExtraArgs(args ...string) {
	helm.extra = args
}

func (helm *execer) SetHelmBinary(bin string) {
	helm.helmBinary = bin
}

func (helm *execer) AddRepo(name, repository, certfile, keyfile, username, password string) error {
	var args []string
	args = append(args, "repo", "add", name, repository)
	if certfile != "" && keyfile != "" {
		args = append(args, "--cert-file", certfile, "--key-file", keyfile)
	}
	if username != "" && password != "" {
		args = append(args, "--username", username, "--password", password)
	}
	helm.logger.Infof("Adding repo %v %v", name, repository)
	out, err := helm.exec(args...)
	helm.write(out)
	return err
}

func (helm *execer) UpdateRepo() error {
	helm.logger.Info("Updating repo")
	out, err := helm.exec("repo", "update")
	helm.write(out)
	return err
}

func (helm *execer) UpdateDeps(chart string) error {
	helm.logger.Infof("Updating dependency %v", chart)
	out, err := helm.exec("dependency", "update", chart)
	helm.write(out)
	return err
}

func (helm *execer) SyncRelease(name, chart string, flags ...string) error {
	helm.logger.Infof("Upgrading %v", chart)
	out, err := helm.exec(append([]string{"upgrade", "--install", "--reset-values", name, chart}, flags...)...)
	helm.write(out)
	return err
}

func (helm *execer) ReleaseStatus(name string) error {
	helm.logger.Infof("Getting status %v", name)
	out, err := helm.exec(append([]string{"status", name})...)
	helm.write(out)
	return err
}

func (helm *execer) DecryptSecret(name string) (string, error) {
	helm.logger.Infof("Decrypting secret %v", name)
	out, err := helm.exec(append([]string{"secrets", "dec", name})...)
	helm.write(out)
	return name + ".dec", err
}

func (helm *execer) DiffRelease(name, chart string, flags ...string) error {
	helm.logger.Infof("Comparing %v %v", name, chart)
	out, err := helm.exec(append([]string{"diff", "upgrade", "--allow-unreleased", name, chart}, flags...)...)
	helm.write(out)
	return err
}

func (helm *execer) Lint(chart string, flags ...string) error {
	helm.logger.Infof("Linting %v", chart)
	out, err := helm.exec(append([]string{"lint", chart}, flags...)...)
	helm.write(out)
	return err
}

func (helm *execer) Fetch(chart string, flags ...string) error {
	helm.logger.Infof("Fetching %v", chart)
	out, err := helm.exec(append([]string{"fetch", chart}, flags...)...)
	helm.write(out)
	return err
}

func (helm *execer) DeleteRelease(name string, flags ...string) error {
	helm.logger.Infof("Deleting %v", name)
	out, err := helm.exec(append([]string{"delete", name}, flags...)...)
	helm.write(out)
	return err
}

func (helm *execer) TestRelease(name string, flags ...string) error {
	helm.logger.Infof("Testing %v", name)
	out, err := helm.exec(append([]string{"test", name}, flags...)...)
	helm.write(out)
	return err
}

func (helm *execer) exec(args ...string) ([]byte, error) {
	cmdargs := args
	if len(helm.extra) > 0 {
		cmdargs = append(cmdargs, helm.extra...)
	}
	if helm.kubeContext != "" {
		cmdargs = append(cmdargs, "--kube-context", helm.kubeContext)
	}
	helm.logger.Debugf("exec: %s %s", helm.helmBinary, strings.Join(cmdargs, " "))
	return helm.runner.Execute(helm.helmBinary, cmdargs)
}

func (helm *execer) write(out []byte) {
	if len(out) > 0 {
		helm.logger.Infof("%s", out)
	}
}
