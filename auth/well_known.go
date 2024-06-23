package auth

import (
	"os"
	v1 "overseer/build/go"
	"overseer/common"
)

// hack: this is a workaround for mostly making grpcui work during development, maybe a better solution to deriving this value and if this ought to be enabled ever outside of local dev
var reflectionMethods = []string{
	"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo",
}
var systemHostname string

const systemTokenKey = "x-auth-system"

func init() {
	hname, err := os.Hostname()
	if err != nil {
		common.GetLogger("auth.init").Error("failed to retrieve hostname", "error", err)
		systemHostname = "unknown"
	}
	systemHostname = hname
}

const SystemUserId = "system"
const SystemActorId = "system"

var systemContextInformation = &common.OverseerContextInformation{
	User: &v1.User{
		Uid: "system",
	},
	Actor: &v1.Actor{
		Uid:            "system",
		SourceIdentity: "system",
		Source:         v1.Actor_SYSTEM,
		Metadata: &v1.ActorMetadata{
			Value: &v1.ActorMetadata_System{
				System: &v1.ActorSourceSystem{
					InstanceId: systemHostname,
				},
			},
		},
	},
}
