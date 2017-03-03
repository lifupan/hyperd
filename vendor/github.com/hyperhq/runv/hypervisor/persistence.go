package hypervisor

import (
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	hyperstartapi "github.com/hyperhq/runv/hyperstart/api/json"
	"github.com/hyperhq/runv/hypervisor/types"
)

type PersistVolumeInfo struct {
	Name       string
	Filename   string
	Format     string
	Fstype     string
	DeviceName string
	ScsiId     int
	Containers []int
	MontPoints []string
}

type PersistNetworkInfo struct {
	Index      int
	PciAddr    int
	DeviceName string
	IpAddr     string
}

type PersistInfo struct {
	Id          string
	DriverInfo  map[string]interface{}
	VmSpec      *hyperstartapi.Pod
	HwStat      *VmHwStatus
	VolumeList  []*PersistVolumeInfo
	NetworkList []*PersistNetworkInfo
}

func (ctx *VmContext) dump() (*PersistInfo, error) {
	dr, err := ctx.DCtx.Dump()
	if err != nil {
		return nil, err
	}

	info := &PersistInfo{
		Id:         ctx.Id,
		DriverInfo: dr,
		//UserSpec:    ctx.userSpec,
		//VmSpec:      ctx.vmSpec,
		HwStat: ctx.dumpHwInfo(),
		//VolumeList:  make([]*PersistVolumeInfo, len(ctx.devices.imageMap)+len(ctx.devices.volumeMap)),
		//NetworkList: make([]*PersistNetworkInfo, len(ctx.devices.networkMap)),
	}

	//vid := 0
	//for _, image := range ctx.devices.imageMap {
	//	info.VolumeList[vid] = image.info.dump()
	//	info.VolumeList[vid].Containers = []int{image.pos}
	//	info.VolumeList[vid].MontPoints = []string{"/"}
	//	vid++
	//}
	//
	//for _, vol := range ctx.devices.volumeMap {
	//	info.VolumeList[vid] = vol.info.dump()
	//	mps := len(vol.pos)
	//	info.VolumeList[vid].Containers = make([]int, mps)
	//	info.VolumeList[vid].MontPoints = make([]string, mps)
	//	i := 0
	//	for idx, mp := range vol.pos {
	//		info.VolumeList[vid].Containers[i] = idx
	//		info.VolumeList[vid].MontPoints[i] = mp
	//		i++
	//	}
	//	vid++
	//}
	//
	//nid := 0
	//for _, nic := range ctx.devices.networkMap {
	//	info.NetworkList[nid] = &PersistNetworkInfo{
	//		Index:      nic.Index,
	//		PciAddr:    nic.PCIAddr,
	//		DeviceName: nic.DeviceName,
	//		IpAddr:     nic.IpAddr,
	//	}
	//	nid++
	//}

	return info, nil
}

func (ctx *VmContext) dumpHwInfo() *VmHwStatus {
	return &VmHwStatus{
		PciAddr:  ctx.pciAddr,
		ScsiId:   ctx.scsiId,
		AttachId: ctx.hyperstart.LastStreamSeq(),
		GuestCid: ctx.GuestCid,
	}
}

func (ctx *VmContext) loadHwStatus(pinfo *PersistInfo) error {
	ctx.pciAddr = pinfo.HwStat.PciAddr
	ctx.scsiId = pinfo.HwStat.ScsiId
	ctx.GuestCid = pinfo.HwStat.GuestCid
	if ctx.GuestCid != 0 {
		if !VsockCidManager.MarkCidInuse(ctx.GuestCid) {
			return fmt.Errorf("conflicting vsock guest cid %d: already in use", ctx.GuestCid)
		}
		ctx.Boot.EnableVsock = true
	}
	return nil
}

func (blk *DiskDescriptor) dump() *PersistVolumeInfo {
	return &PersistVolumeInfo{
		Name:       blk.Name,
		Filename:   blk.Filename,
		Format:     blk.Format,
		Fstype:     blk.Fstype,
		DeviceName: blk.DeviceName,
		ScsiId:     blk.ScsiId,
	}
}

func (vol *PersistVolumeInfo) blockInfo() *DiskDescriptor {
	return &DiskDescriptor{
		Name:       vol.Name,
		Filename:   vol.Filename,
		Format:     vol.Format,
		Fstype:     vol.Fstype,
		DeviceName: vol.DeviceName,
		ScsiId:     vol.ScsiId,
	}
}

func vmDeserialize(s []byte) (*PersistInfo, error) {
	info := &PersistInfo{}
	err := json.Unmarshal(s, info)
	return info, err
}

func (pinfo *PersistInfo) serialize() ([]byte, error) {
	return json.Marshal(pinfo)
}

func (pinfo *PersistInfo) vmContext(hub chan VmEvent, client chan *types.VmResponse) (*VmContext, error) {

	dc, err := HDriver.LoadContext(pinfo.DriverInfo)
	if err != nil {
		glog.Error("cannot load driver context: ", err.Error())
		return nil, err
	}

	ctx, err := InitContext(pinfo.Id, hub, client, dc, &BootConfig{})
	if err != nil {
		return nil, err
	}

	//ctx.vmSpec = pinfo.VmSpec
	//ctx.userSpec = pinfo.UserSpec
	//ctx.wg = wg

	err = ctx.loadHwStatus(pinfo)
	if err != nil {
		return nil, err
	}

	//for _, vol := range pinfo.VolumeList {
	//	binfo := vol.blockInfo()
	//	if len(vol.Containers) != len(vol.MontPoints) {
	//		return nil, errors.New("persistent data corrupt, volume info mismatch")
	//	}
	//	if len(vol.MontPoints) == 1 && vol.MontPoints[0] == "/" {
	//		img := &imageInfo{
	//			info: binfo,
	//			pos:  vol.Containers[0],
	//		}
	//		ctx.devices.imageMap[vol.Name] = img
	//	} else {
	//		v := &volume{
	//			info:     binfo,
	//			pos:      make(map[int]string),
	//			readOnly: make(map[int][]bool),
	//		}
	//		for i := 0; i < len(vol.Containers); i++ {
	//			idx := vol.Containers[i]
	//			v.pos[idx] = vol.MontPoints[i]
	//			v.readOnly[idx] = ctx.vmSpec.Containers[idx].RoLookup(vol.MontPoints[i])
	//		}
	//		ctx.devices.volumeMap[vol.Name] = v
	//	}
	//}
	//
	//for _, nic := range pinfo.NetworkList {
	//	ctx.devices.networkMap[nic.Index] = &InterfaceCreated{
	//		Index:      nic.Index,
	//		PCIAddr:    nic.PciAddr,
	//		DeviceName: nic.DeviceName,
	//		IpAddr:     nic.IpAddr,
	//	}
	//}

	return ctx, nil
}
