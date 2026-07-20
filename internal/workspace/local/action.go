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

type WorkspacePlanAction string

const (
	WorkspacePlanActionCreate   WorkspacePlanAction = "create"
	WorkspacePlanActionStart    WorkspacePlanAction = "start"
	WorkspacePlanActionReuse    WorkspacePlanAction = "reuse"
	WorkspacePlanActionRecreate WorkspacePlanAction = "recreate"
	WorkspacePlanActionRemove   WorkspacePlanAction = "remove"
	WorkspacePlanActionCleanup  WorkspacePlanAction = "cleanup"
	WorkspacePlanActionNone     WorkspacePlanAction = "none"
)

type WorkspaceChangeReason string

const (
	WorkspaceChangeReasonAbsent             WorkspaceChangeReason = "workspace_absent"
	WorkspaceChangeReasonStopped            WorkspaceChangeReason = "workspace_stopped"
	WorkspaceChangeReasonExplicitRecreate   WorkspaceChangeReason = "explicit_recreate"
	WorkspaceChangeReasonEnvironment        WorkspaceChangeReason = "environment_changed"
	WorkspaceChangeReasonImage              WorkspaceChangeReason = "image_changed"
	WorkspaceChangeReasonPlatform           WorkspaceChangeReason = "platform_changed"
	WorkspaceChangeReasonHostname           WorkspaceChangeReason = "hostname_changed"
	WorkspaceChangeReasonPublishedPorts     WorkspaceChangeReason = "published_ports_changed"
	WorkspaceChangeReasonMounts             WorkspaceChangeReason = "mounts_changed"
	WorkspaceChangeReasonPrivileged         WorkspaceChangeReason = "privileged_changed"
	WorkspaceChangeReasonRuntimeEnvironment WorkspaceChangeReason = "runtime_environment_changed"
)

type WorkspaceImageStatus string

const (
	WorkspaceImageStatusAvailable    WorkspaceImageStatus = "available"
	WorkspaceImageStatusPullRequired WorkspaceImageStatus = "pull_required"
)

type WorkspaceManagedResource string

const (
	WorkspaceManagedResourceContainerWritableLayer WorkspaceManagedResource = "container_writable_layer"
	WorkspaceManagedResourceManagedSSH             WorkspaceManagedResource = "managed_ssh"
	WorkspaceManagedResourceKnownHosts             WorkspaceManagedResource = "known_hosts"
	WorkspaceManagedResourceRegistryRecord         WorkspaceManagedResource = "registry_record"
)

type WorkspacePreservedResource string

const (
	WorkspacePreservedResourceProjectFiles    WorkspacePreservedResource = "project_files"
	WorkspacePreservedResourceMountedHostData WorkspacePreservedResource = "mounted_host_data"
	WorkspacePreservedResourceLocalImages     WorkspacePreservedResource = "local_images"
)
