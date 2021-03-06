package uvm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Microsoft/hcsshim/internal/guestrequest"
	"github.com/Microsoft/hcsshim/internal/requesttype"
	hcsschema "github.com/Microsoft/hcsshim/internal/schema2"
)

// Share shares in file(s) from `reqHostPath` on the host machine to `reqUVMPath` inside the UVM.
// This function handles both LCOW and WCOW scenarios.
func (uvm *UtilityVM) Share(ctx context.Context, reqHostPath, reqUVMPath string, readOnly bool) (err error) {
	if uvm.OS() == "windows" {
		options := uvm.DefaultVSMBOptions(readOnly)
		vsmbShare, err := uvm.AddVSMB(ctx, reqHostPath, options)
		if err != nil {
			return err
		}
		defer func() {
			if err != nil {
				vsmbShare.Release(ctx) //nolint:errcheck
			}
		}()

		sharePath, err := uvm.GetVSMBUvmPath(ctx, reqHostPath, readOnly)
		if err != nil {
			return err
		}
		guestReq := guestrequest.GuestRequest{
			ResourceType: guestrequest.ResourceTypeMappedDirectory,
			RequestType:  requesttype.Add,
			Settings: &hcsschema.MappedDirectory{
				HostPath:      sharePath,
				ContainerPath: reqUVMPath,
				ReadOnly:      readOnly,
			},
		}
		if err := uvm.GuestRequest(ctx, guestReq); err != nil {
			return err
		}
	} else {
		st, err := os.Stat(reqHostPath)
		if err != nil {
			return fmt.Errorf("could not open '%s' path on host: %s", reqHostPath, err)
		}
		var (
			hostPath       string = reqHostPath
			restrictAccess bool
			fileName       string
			allowedNames   []string
		)
		if !st.IsDir() {
			hostPath, fileName = filepath.Split(hostPath)
			allowedNames = append(allowedNames, fileName)
			restrictAccess = true
		}
		plan9Share, err := uvm.AddPlan9(ctx, hostPath, reqUVMPath, readOnly, restrictAccess, allowedNames)
		if err != nil {
			return err
		}
		defer func() {
			if err != nil {
				plan9Share.Release(ctx) //nolint:errcheck
			}
		}()
	}
	return nil
}
