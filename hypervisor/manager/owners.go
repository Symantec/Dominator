package manager

import (
	"github.com/Symantec/Dominator/lib/srpc"
)

func (m *Manager) changeOwners(ownerGroups, ownerUsers []string) error {
	ownerGroupsMap := make(map[string]struct{}, len(ownerGroups))
	for _, group := range ownerGroups {
		ownerGroupsMap[group] = struct{}{}
	}
	ownerUsersMap := make(map[string]struct{}, len(ownerUsers))
	for _, user := range ownerUsers {
		ownerUsersMap[user] = struct{}{}
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.ownerGroups = ownerGroupsMap
	m.ownerUsers = ownerUsersMap
	return nil
}

func (m *Manager) checkOwnership(authInfo *srpc.AuthInformation) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if authInfo.Username != "" {
		if _, ok := m.ownerUsers[authInfo.Username]; ok {
			return true
		}
	}
	for group := range authInfo.GroupList {
		if _, ok := m.ownerGroups[group]; ok {
			return true
		}
	}
	return false
}
