package local

type WorkspaceAction string

const (
	WorkspaceActionCreated   WorkspaceAction = "created"
	WorkspaceActionRecreated WorkspaceAction = "recreated"
	WorkspaceActionStarted   WorkspaceAction = "started"
	WorkspaceActionReused    WorkspaceAction = "reused"
)

func (action WorkspaceAction) CreatedContainer() bool {
	return action == WorkspaceActionCreated || action == WorkspaceActionRecreated
}
