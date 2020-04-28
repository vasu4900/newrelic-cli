package cliplugin

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"os"
	"os/exec"

	hclog "github.com/hashicorp/go-hclog"
	plugin "github.com/hashicorp/go-plugin"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	proto "github.com/newrelic/newrelic-cli/internal/plugins/protoDef"
)

const (
	pluginType       = "cli_plugin"
	magicCookieKey   = "NEWRELIC_CLI_PLUGIN"
	magicCookieValue = "4951e1a8-27fa-4fc0-b04c-308fc3ed5799"
)

var (
	handshakeConfig = plugin.HandshakeConfig{
		ProtocolVersion:  1,
		MagicCookieKey:   magicCookieKey,
		MagicCookieValue: magicCookieValue,
	}
	pluginMap = map[string]plugin.Plugin{
		pluginType: &CLIPlugin{},
	}
)

// Client is used for communicating with a CLI plugin.
type Client struct {
	pb         proto.CLIClient
	pluginHost *plugin.Client
}

// ClientOptions represents the options to be passed to the client.
type ClientOptions struct {
	LogLevel string
	Command  string
	Args     []string
}

// NewClient creates a new client for communicating with a CLI plugin.
func NewClient(opts *ClientOptions) *Client {
	if opts.LogLevel == "" {
		opts.LogLevel = "Info"
	}

	logger := hclog.New(&hclog.LoggerOptions{
		Level: hclog.LevelFromString(opts.LogLevel),
	})

	pluginHost := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  handshakeConfig,
		Plugins:          pluginMap,
		Cmd:              exec.Command(opts.Command, opts.Args...),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Logger:           logger,
	})

	rpcClient, err := pluginHost.Client()
	if err != nil {
		log.Fatalf(err.Error())
	}

	raw, err := rpcClient.Dispense(pluginType)
	if err != nil {
		log.Fatalf(err.Error())
	}

	client := raw.(*Client)
	client.pluginHost = pluginHost

	return client
}

// Kill ends the plugin host process and cleans up remaining resources.
func (c *Client) Kill() {
	c.pluginHost.Kill()
}

// Exec allows for executing a given subcommand.
func (c *Client) Exec(command string, args []string) error {
	stream, err := c.pb.Exec(context.Background())

	stream.Send(&proto.ExecRequest{
		Command: command,
		Args:    args,
	})

	if err != nil {
		return err
	}

	var stdout, stderr bytes.Buffer

	go handleStdin(stream)

	for {
		chunk, err := stream.Recv()

		if err == io.EOF {
			break
		}

		if err != nil {
			break
		}

		stdout.Write(chunk.Stdout)
		_, err = io.Copy(os.Stdout, &stdout)
		if err != nil {
			log.Fatal(err)
		}

		stderr.Write(chunk.Stderr)
		_, err = io.Copy(os.Stderr, &stderr)
		if err != nil {
			log.Fatal(err)
		}
	}

	return nil
}

func handleStdin(stream proto.CLI_ExecClient) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		scanner.Scan()
		stream.Send(&proto.ExecRequest{
			Stdin: scanner.Bytes(),
		})
	}
}

// CLIPlugin represents a gRPC-aware plugin powered by go-plugin.
// It satisfies the plugin.GRPCPlugin interface.
type CLIPlugin struct {
	plugin.Plugin
}

// GRPCServer creates a gRPC server for running a plugin.
// This is currently not implemented, but is here to satisfy the underlying interface.
func (p *CLIPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	return nil
}

// GRPCClient creates a gRPC client for communicating with a plugin.
func (p *CLIPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &Client{pb: proto.NewCLIClient(c)}, nil
}
