// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2021 Intel Corporation

package daemon

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/sirupsen/logrus"

	sriovv2 "github.com/smart-edge-open/sriov-fec-operator/sriov-fec/api/v2"
	"gopkg.in/ini.v1"
)

const (
	ul           = "UL"
	dl           = "DL"
	flr          = "FLR"
	bandwidth    = "bandwidth"
	load_balance = "load_balance"
	vfqmap       = "vfqmap"
	flr_time_out = "flr_time_out"

	vfbundles          = "VFBUNDLES"
	maxqsize           = "MAXQSIZE"
	uplink4g           = "QUL4G"
	downlink4g         = "QDL4G"
	uplink5g           = "QUL5G"
	downlink5g         = "QDL5G"
	num_vf_bundles     = "num_vf_bundles"
	max_queue_size     = "max_queue_size"
	num_qgroups        = "num_qgroups"
	num_aqs_per_groups = "num_aqs_per_groups"
	aq_depth_log2      = "aq_depth_log2"
	maxQueueGroups     = 8

	mode                = "MODE"
	pf_mode_en          = "pf_mode_en"
	pfConfigAppFilepath = "/sriov_workdir/pf_bb_config"
)

func generateN3000BBDevConfigFile(log *logrus.Logger, nc *sriovv2.N3000BBDevConfig, file string) error {
	if nc == nil {
		return errors.New("received nil N3000BBDevConfig")
	}

	cfg := ini.Empty()
	err := cfg.NewSections(mode, ul, dl, flr)
	if err != nil {
		return fmt.Errorf("Unable to create sections in bbdevconfig")
	}

	var modeValue string
	if nc.PFMode {
		modeValue = "1"
	} else {
		modeValue = "0"
	}
	cfg.Section(mode).Key(pf_mode_en).SetValue(modeValue)
	cfg.Section(ul).Key(bandwidth).SetValue(strconv.Itoa(nc.Uplink.Bandwidth))
	cfg.Section(ul).Key(load_balance).SetValue(strconv.Itoa(nc.Uplink.LoadBalance))
	cfg.Section(ul).Key(vfqmap).SetValue(nc.Uplink.Queues.String())
	cfg.Section(dl).Key(bandwidth).SetValue(strconv.Itoa(nc.Downlink.Bandwidth))
	cfg.Section(dl).Key(load_balance).SetValue(strconv.Itoa(nc.Downlink.LoadBalance))
	cfg.Section(dl).Key(vfqmap).SetValue(nc.Downlink.Queues.String())
	cfg.Section(flr).Key(flr_time_out).SetValue(strconv.Itoa(nc.FLRTimeOut))

	err = logBBDevConfigFile(log, cfg)
	if err != nil {
		return err
	}

	err = cfg.SaveTo(file)
	if err != nil {
		return fmt.Errorf("Unable to write config to file: %s", file)
	}
	return nil
}

func generateACC100BBDevConfigFile(log *logrus.Logger, nc *sriovv2.ACC100BBDevConfig, file string) error {
	if nc == nil {
		return errors.New("received nil ACC100BBDevConfig")
	}

	total4GQueueGroups := nc.Uplink4G.NumQueueGroups + nc.Downlink4G.NumQueueGroups
	total5GQueueGroups := nc.Uplink5G.NumQueueGroups + nc.Downlink5G.NumQueueGroups
	totalQueueGroups := total4GQueueGroups + total5GQueueGroups
	if totalQueueGroups > maxQueueGroups {
		return fmt.Errorf("Total number of requested queue groups (4G/5G) exceeds the maximum (%d)", maxQueueGroups)
	}

	cfg := ini.Empty()
	err := cfg.NewSections(mode, vfbundles, maxqsize, uplink4g, downlink4g, uplink5g, downlink5g)
	if err != nil {
		return fmt.Errorf("Unable to create sections in bbdevconfig")
	}

	var modeValue string
	if nc.PFMode {
		modeValue = "1"
	} else {
		modeValue = "0"
	}
	cfg.Section(mode).Key(pf_mode_en).SetValue(modeValue)
	cfg.Section(vfbundles).Key(num_vf_bundles).SetValue(strconv.Itoa(nc.NumVfBundles))
	cfg.Section(maxqsize).Key(max_queue_size).SetValue(strconv.Itoa(nc.MaxQueueSize))
	cfg.Section(uplink4g).Key(num_qgroups).SetValue(strconv.Itoa(nc.Uplink4G.NumQueueGroups))
	cfg.Section(uplink4g).Key(num_aqs_per_groups).SetValue(strconv.Itoa(nc.Uplink4G.NumAqsPerGroups))
	cfg.Section(uplink4g).Key(aq_depth_log2).SetValue(strconv.Itoa(nc.Uplink4G.AqDepthLog2))
	cfg.Section(downlink4g).Key(num_qgroups).SetValue(strconv.Itoa(nc.Downlink4G.NumQueueGroups))
	cfg.Section(downlink4g).Key(num_aqs_per_groups).SetValue(strconv.Itoa(nc.Downlink4G.NumAqsPerGroups))
	cfg.Section(downlink4g).Key(aq_depth_log2).SetValue(strconv.Itoa(nc.Downlink4G.AqDepthLog2))
	cfg.Section(uplink5g).Key(num_qgroups).SetValue(strconv.Itoa(nc.Uplink5G.NumQueueGroups))
	cfg.Section(uplink5g).Key(num_aqs_per_groups).SetValue(strconv.Itoa(nc.Uplink5G.NumAqsPerGroups))
	cfg.Section(uplink5g).Key(aq_depth_log2).SetValue(strconv.Itoa(nc.Uplink5G.AqDepthLog2))
	cfg.Section(downlink5g).Key(num_qgroups).SetValue(strconv.Itoa(nc.Downlink5G.NumQueueGroups))
	cfg.Section(downlink5g).Key(num_aqs_per_groups).SetValue(strconv.Itoa(nc.Downlink5G.NumAqsPerGroups))
	cfg.Section(downlink5g).Key(aq_depth_log2).SetValue(strconv.Itoa(nc.Downlink5G.AqDepthLog2))

	err = logBBDevConfigFile(log, cfg)
	if err != nil {
		return err
	}

	err = cfg.SaveTo(file)
	if err != nil {
		return fmt.Errorf("Unable to write config to file: %s", file)
	}
	return nil
}

func generateBBDevConfigFile(log *logrus.Logger, pfCfg sriovv2.BBDevConfig, file string) error {
	if pfCfg.ACC100 != nil {
		if err := generateACC100BBDevConfigFile(log, pfCfg.ACC100, file); err != nil {
			return fmt.Errorf("ACC100 config file creation failed, %s", err)
		}
	} else if pfCfg.N3000 != nil {
		if err := generateN3000BBDevConfigFile(log, pfCfg.N3000, file); err != nil {
			return fmt.Errorf("N3000 config file creation failed, %s", err)
		}
	} else {
		return fmt.Errorf("Received nil configs")
	}

	return nil
}

// runPFConfig executes a pf-bb-config tool
// deviceName is one of: FPGA_LTE or FPGA_5GNR or ACC100
// cfgFilepath is a filepath to the config
// pciAddress points to a specific PF device
func runPFConfig(log *logrus.Logger, deviceName, cfgFilepath, pciAddress string) error {
	switch deviceName {
	case "FPGA_LTE", "FPGA_5GNR", "ACC100":
	default:
		return fmt.Errorf("incorrect deviceName for pf config: %s", deviceName)
	}
	_, err := runExecCmd([]string{pfConfigAppFilepath, deviceName, "-c", cfgFilepath, "-p", pciAddress}, log)
	return err
}

func logBBDevConfigFile(log *logrus.Logger, cfg *ini.File) error {
	var b bytes.Buffer
	writer := io.Writer(&b)
	_, err := cfg.WriteTo(writer)
	if err != nil {
		return fmt.Errorf("Unable to write config to log writer, %s", err)
	}
	log.WithField("generated BBDevConfig", b.String()).Info("logBBDevConfigFile")
	return nil
}
