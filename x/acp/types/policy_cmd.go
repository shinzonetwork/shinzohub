package types

func NewSetRelationshipCmd(rel *Relationship) *PolicyCmd {
	return &PolicyCmd{
		Cmd: &PolicyCmd_SetRelationshipCmd{
			SetRelationshipCmd: &SetRelationshipCmd{
				Relationship: rel,
			},
		},
	}
}

func NewDeleteRelationshipCmd(rel *Relationship) *PolicyCmd {
	return &PolicyCmd{
		Cmd: &PolicyCmd_DeleteRelationshipCmd{
			DeleteRelationshipCmd: &DeleteRelationshipCmd{
				Relationship: rel,
			},
		},
	}
}

func NewRegisterObjectCmd(obj *Object) *PolicyCmd {
	return &PolicyCmd{
		Cmd: &PolicyCmd_RegisterObjectCmd{
			RegisterObjectCmd: &RegisterObjectCmd{
				Object: obj,
			},
		},
	}
}

func NewUnregisterObjectCmd(obj *Object) *PolicyCmd {
	return &PolicyCmd{
		Cmd: &PolicyCmd_UnregisterObjectCmd{
			UnregisterObjectCmd: &UnregisterObjectCmd{
				Object: obj,
			},
		},
	}
}
