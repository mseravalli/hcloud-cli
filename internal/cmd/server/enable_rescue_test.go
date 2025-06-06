package server_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/hetznercloud/cli/internal/cmd/server"
	"github.com/hetznercloud/cli/internal/testutil"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

func TestEnableRescue(t *testing.T) {
	fx := testutil.NewFixture(t)
	defer fx.Finish()

	cmd := server.EnableRescueCmd.CobraCommand(fx.State())
	fx.ExpectEnsureToken()

	srv := &hcloud.Server{ID: 123, Name: "my-server"}

	fx.Client.ServerClient.EXPECT().
		Get(gomock.Any(), "my-server").
		Return(srv, nil, nil)
	fx.Client.ServerClient.EXPECT().
		EnableRescue(gomock.Any(), srv, hcloud.ServerEnableRescueOpts{Type: hcloud.ServerRescueTypeLinux64}).
		Return(hcloud.ServerEnableRescueResult{
			Action:       &hcloud.Action{ID: 789},
			RootPassword: "root-password",
		}, nil, nil)
	fx.ActionWaiter.EXPECT().
		WaitForActions(gomock.Any(), gomock.Any(), &hcloud.Action{ID: 789}).
		Return(nil)

	args := []string{"my-server"}
	out, errOut, err := fx.Run(cmd, args)

	require.NoError(t, err)
	assert.Empty(t, errOut)
	assert.Equal(t, "Rescue enabled for Server 123 with root password: root-password\n", out)
}
