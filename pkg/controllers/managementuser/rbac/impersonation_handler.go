package rbac

import (
	"github.com/rancher/rancher/pkg/impersonation"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/authentication/user"
)

func (m *manager) getUser(username, groupname string) (user.Info, error) {
	u, err := m.userLister.Get("", username)
	if err != nil {
		return &user.DefaultInfo{}, err
	}
	groups := []string{"system:authenticated", "system:cattle:authenticated"}
	if groupname != "" {
		groups = append(groups, groupname)
	}
	attribs, err := m.userAttributeLister.Get("", username)
	if err != nil && !apierrors.IsNotFound(err) {
		return &user.DefaultInfo{}, err
	}
	if attribs != nil {
		for _, gps := range attribs.GroupPrincipals {
			for _, principal := range gps.Items {
				groups = append(groups, principal.Name)
			}
		}
	}
	user := &user.DefaultInfo{
		UID:    u.GetName(),
		Name:   u.Username,
		Groups: groups,
		Extra:  map[string][]string{"username": []string{u.Username}},
	}
	if len(u.PrincipalIDs) > 0 {
		user.Extra["principalid"] = u.PrincipalIDs
	}
	return user, nil
}

func (m *manager) ensureServiceAccountImpersonator(username, groupname string) error {
	user, err := m.getUser(username, groupname)
	if apierrors.IsNotFound(err) {
		logrus.Warnf("could not find user %s, will not create impersonation account on cluster", username)
		return nil
	}
	if err != nil {
		return err
	}
	logrus.Debugf("ensuring service account impersonator for %s", user.GetUID())
	i := impersonation.New(user, m.workload)
	_, err = i.SetUpImpersonation()
	return err
}

func (m *manager) deleteServiceAccountImpersonator(username string) error {
	crtbs, err := m.crtbIndexer.ByIndex(rtbByClusterAndUserIndex, m.workload.ClusterName+"-"+username)
	if err != nil {
		return err
	}
	prtbs, err := m.prtbIndexer.ByIndex(rtbByClusterAndUserIndex, m.workload.ClusterName+"-"+username)
	if err != nil {
		return err
	}
	if len(crtbs)+len(prtbs) > 0 {
		return nil
	}
	roleName := impersonation.ImpersonationPrefix + username
	logrus.Debugf("deleting service account impersonator for %s", username)
	err = m.workload.RBAC.ClusterRoles("").Delete(roleName, &metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}
