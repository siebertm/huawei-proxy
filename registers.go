package main

import "fmt"

// RegisterDef describes a single Modbus register entry.
type RegisterDef struct {
	Name   string
	Length uint16
}

// registerDefs maps register start addresses to their definitions.
// Extracted from REGISTERS.md (huawei_solar v2.5.0).
var registerDefs = map[uint16]RegisterDef{
	// Device Information (30000–31210)
	30000: {"model_name", 15},
	30015: {"serial_number", 10},
	30025: {"pn", 10},
	30035: {"firmware_version", 15},
	30050: {"software_version", 15},
	30068: {"protocol_version_modbus", 2},
	30070: {"model_id", 1},
	30071: {"nb_pv_strings", 1},
	30072: {"nb_mpp_tracks", 1},
	30073: {"rated_power", 2},
	30075: {"P_max", 2},
	30077: {"S_max", 2},
	30079: {"Q_max_out", 2},
	30081: {"Q_max_in", 2},
	30083: {"P_max_real", 2},
	30085: {"S_max_real", 2},
	30105: {"product_sales_area", 2},
	30107: {"product_software_number", 1},
	30108: {"product_software_version_number", 1},
	30109: {"grid_standard_code_protocol_version", 1},
	30110: {"unique_id_of_the_software", 1},
	30111: {"number_of_packages_to_be_upgraded", 1},
	30206: {"hardware_functional_unit_conf_id", 1},
	30207: {"subdevice_support_flag", 2},
	30209: {"subdevice_in_position_flag", 2},
	30211: {"feature_mask_1", 2},
	30213: {"feature_mask_2", 2},
	30215: {"feature_mask_3", 2},
	30217: {"feature_mask_4", 2},
	30366: {"realtime_max_active_capability", 2},
	30368: {"realtime_max_inductive_reactive_capacity", 2},
	30561: {"offering_name_of_southbound_device_1", 15},
	30576: {"offering_name_of_southbound_device_2", 15},
	30591: {"offering_name_of_southbound_device_3", 15},
	31000: {"hardware_version", 15},
	31015: {"monitoring_board_sn", 10},
	31025: {"monitoring_software_version", 15},
	31040: {"master_dsp_version", 15},
	31055: {"slave_dsp_version", 15},
	31070: {"cpld_version", 15},
	31085: {"afci_version", 15},
	31100: {"builtin_pid_version", 15},
	31115: {"dc_mbus_version", 15},
	31130: {"el_module_version", 15},
	31145: {"afci_2_version", 15},
	31200: {"regkey", 10},

	// Inverter State & Alarms (32000–32010)
	32000: {"state_1", 1},
	32002: {"state_2", 1},
	32003: {"state_3", 2},
	32008: {"alarm_1", 1},
	32009: {"alarm_2", 1},
	32010: {"alarm_3", 1},

	// PV String Data (32016–32063) — generated in init()

	// Grid Output & Power (32064–32097)
	32064: {"input_power", 2},
	32066: {"line_voltage_A_B", 1},
	32067: {"line_voltage_B_C", 1},
	32068: {"line_voltage_C_A", 1},
	32069: {"phase_A_voltage", 1},
	32070: {"phase_B_voltage", 1},
	32071: {"phase_C_voltage", 1},
	32072: {"phase_A_current", 2},
	32074: {"phase_B_current", 2},
	32076: {"phase_C_current", 2},
	32078: {"day_active_power_peak", 2},
	32080: {"active_power", 2},
	32082: {"reactive_power", 2},
	32084: {"power_factor", 1},
	32085: {"grid_frequency", 1},
	32086: {"efficiency", 1},
	32087: {"internal_temperature", 1},
	32088: {"insulation_resistance", 1},
	32089: {"device_status", 1},
	32090: {"fault_code", 1},
	32091: {"startup_time", 2},
	32093: {"shutdown_time", 2},
	32095: {"active_power_fast", 2},

	// Energy Yield (32106–32120)
	32106: {"accumulated_yield_energy", 2},
	32108: {"total_dc_input_power", 2},
	32110: {"current_electricity_generation_statistics_time", 2},
	32112: {"hourly_yield_energy", 2},
	32114: {"daily_yield_energy", 2},
	32116: {"monthly_yield_energy", 2},
	32118: {"yearly_yield_energy", 2},

	// Diagnostic (32172–32191)
	32172: {"latest_active_alarm_sn", 2},
	32174: {"latest_historical_alarm_sn", 2},
	32176: {"total_bus_voltage", 1},
	32177: {"max_pv_voltage", 1},
	32178: {"min_pv_voltage", 1},
	32179: {"average_pv_negative_voltage_to_ground", 1},
	32180: {"min_pv_negative_voltage_to_ground", 1},
	32181: {"max_pv_negative_voltage_to_ground", 1},
	32182: {"inverter_to_pe_voltage_tolerance", 1},
	32183: {"iso_feature_information", 1},
	32190: {"builtin_pid_running_status", 1},
	32191: {"pv_negative_voltage_to_ground", 1},

	// MPPT cumulative yields (32212–32231) — generated in init()

	// Component Health (35000–35044)
	35000: {"capbank_running_time", 2},
	35002: {"internal_fan_1_running_time", 2},
	35021: {"inv_module_a_temp", 1},
	35022: {"inv_module_b_temp", 1},
	35023: {"inv_module_c_temp", 1},
	35024: {"anti_reverse_module_1_temp", 1},
	35025: {"output_board_relay_ambient_temp_max", 1},
	35027: {"anti_reverse_module_2_temp", 1},
	35028: {"dc_terminal_1_2_max_temp", 1},
	35029: {"ac_terminal_1_2_3_max_temp", 1},
	35038: {"phase_a_dc_component_dci", 1},
	35039: {"phase_b_dc_component_dci", 1},
	35040: {"phase_c_dc_component_dci", 1},
	35041: {"leakage_current_rcd", 1},
	35042: {"positive_bus_voltage", 1},
	35043: {"negative_bus_voltage", 1},
	35044: {"bus_negative_voltage_to_ground", 1},

	// Storage Unit 1 (37000–37068)
	37000: {"storage_unit_1_running_status", 1},
	37001: {"storage_unit_1_charge_discharge_power", 2},
	37003: {"storage_unit_1_bus_voltage", 1},
	37004: {"storage_unit_1_state_of_capacity", 1},
	37006: {"storage_unit_1_working_mode_b", 1},
	37007: {"storage_unit_1_rated_charge_power", 2},
	37009: {"storage_unit_1_rated_discharge_power", 2},
	37014: {"storage_unit_1_fault_id", 1},
	37015: {"storage_unit_1_current_day_charge_capacity", 2},
	37017: {"storage_unit_1_current_day_discharge_capacity", 2},
	37021: {"storage_unit_1_bus_current", 1},
	37022: {"storage_unit_1_battery_temperature", 1},
	37025: {"storage_unit_1_remaining_charge_dis_charge_time", 1},
	37026: {"storage_unit_1_dcdc_version", 10},
	37036: {"storage_unit_1_bms_version", 10},
	37046: {"storage_maximum_charge_power", 2},
	37048: {"storage_maximum_discharge_power", 2},
	37052: {"storage_unit_1_serial_number", 10},
	37066: {"storage_unit_1_total_charge", 2},
	37068: {"storage_unit_1_total_discharge", 2},

	// Power Meter (37100–37139)
	37100: {"meter_status", 1},
	37101: {"grid_A_voltage", 2},
	37103: {"grid_B_voltage", 2},
	37105: {"grid_C_voltage", 2},
	37107: {"active_grid_A_current", 2},
	37109: {"active_grid_B_current", 2},
	37111: {"active_grid_C_current", 2},
	37113: {"power_meter_active_power", 2},
	37115: {"power_meter_reactive_power", 2},
	37117: {"active_grid_power_factor", 1},
	37118: {"active_grid_frequency", 1},
	37119: {"grid_exported_energy", 2},
	37121: {"grid_accumulated_energy", 2},
	37123: {"grid_accumulated_reactive_power", 2},
	37125: {"meter_type", 1},
	37126: {"active_grid_A_B_voltage", 2},
	37128: {"active_grid_B_C_voltage", 2},
	37130: {"active_grid_C_A_voltage", 2},
	37132: {"active_grid_A_power", 2},
	37134: {"active_grid_B_power", 2},
	37136: {"active_grid_C_power", 2},
	37138: {"meter_type_check", 1},

	// Optimizers (37200–37201)
	37200: {"nb_optimizers", 1},
	37201: {"nb_online_optimizers", 1},

	// Storage Unit 2 (37700–37829)
	37700: {"storage_unit_2_serial_number", 10},
	37738: {"storage_unit_2_state_of_capacity", 1},
	37741: {"storage_unit_2_running_status", 1},
	37743: {"storage_unit_2_charge_discharge_power", 2},
	37746: {"storage_unit_2_current_day_charge_capacity", 2},
	37748: {"storage_unit_2_current_day_discharge_capacity", 2},
	37750: {"storage_unit_2_bus_voltage", 1},
	37751: {"storage_unit_2_bus_current", 1},
	37752: {"storage_unit_2_battery_temperature", 1},
	37753: {"storage_unit_2_total_charge", 2},
	37755: {"storage_unit_2_total_discharge", 2},
	37799: {"storage_unit_2_software_version", 15},
	37814: {"storage_unit_1_software_version", 15},

	// Storage Aggregated (37758–37788)
	37758: {"storage_rated_capacity", 2},
	37760: {"storage_state_of_capacity", 1},
	37762: {"storage_running_status", 1},
	37763: {"storage_bus_voltage", 1},
	37764: {"storage_bus_current", 1},
	37765: {"storage_charge_discharge_power", 2},
	37780: {"storage_total_charge", 2},
	37782: {"storage_total_discharge", 2},
	37784: {"storage_current_day_charge_capacity", 2},
	37786: {"storage_current_day_discharge_capacity", 2},

	// Battery Pack SOH (37920–37927)
	37920: {"storage_unit_1_battery_pack_1_soh_calibration_status", 1},
	37921: {"storage_unit_1_battery_pack_2_soh_calibration_status", 1},
	37922: {"storage_unit_1_battery_pack_3_soh_calibration_status", 1},
	37923: {"storage_unit_2_battery_pack_1_soh_calibration_status", 1},
	37924: {"storage_unit_2_battery_pack_2_soh_calibration_status", 1},
	37925: {"storage_unit_2_battery_pack_3_soh_calibration_status", 1},
	37926: {"storage_unit_soh_calibration_status", 1},
	37927: {"storage_unit_soh_calibration_release_lower_limit_of_soc", 1},

	// Battery Pack Details (38200–38451) — generated in init()
	// Battery Pack Temperatures (38452–38463) — generated in init()

	// System Time (40000–40001)
	40000: {"system_time", 2},

	// Inverter Control (40037–40201)
	40037: {"q_u_characteristic_curve_model", 1},
	40038: {"q_u_scheduling_trigger_power_percentage", 1},
	40122: {"power_factor_2", 1},
	40123: {"reactive_power_compensation", 1},
	40124: {"reactive_power_adjustment_time", 1},
	40125: {"active_power_percentage_derating", 1},
	40126: {"active_power_fixed_value_derating", 2},
	40128: {"reactive_power_compensation_at_night", 1},
	40129: {"fixed_reactive_power_at_night", 2},
	40196: {"characteristic_curve_reactive_power_adjustment_time", 1},
	40197: {"percent_apparent_power", 1},
	40198: {"q_u_scheduling_exit_power_percentage", 1},
	40200: {"startup", 1},
	40201: {"shutdown", 1},

	// Grid & MPPT Config (42000–43006)
	42000: {"grid_code", 1},
	42054: {"mppt_multimodal_scanning", 1},
	42055: {"mppt_scanning_interval", 1},
	42056: {"mppt_predicted_power", 2},
	42900: {"daylight_saving_time", 1},
	43006: {"time_zone", 1},

	// Storage Configuration (47000–48020)
	47000: {"storage_unit_1_product_model", 1},
	47004: {"storage_working_mode_a", 1},
	47027: {"storage_time_of_use_price", 1},
	47028: {"storage_time_of_use_price_periods", 41},
	47069: {"storage_lcoe", 2},
	47075: {"storage_maximum_charging_power", 2},
	47077: {"storage_maximum_discharging_power", 2},
	47079: {"storage_power_limit_grid_tied_point", 2},
	47081: {"storage_charging_cutoff_capacity", 1},
	47082: {"storage_discharging_cutoff_capacity", 1},
	47083: {"storage_forced_charging_and_discharging_period", 1},
	47084: {"storage_forced_charging_and_discharging_power", 2},
	47086: {"storage_working_mode_settings", 1},
	47087: {"storage_charge_from_grid_function", 1},
	47088: {"storage_grid_charge_cutoff_state_of_charge", 1},
	47089: {"storage_unit_2_product_model", 1},
	47100: {"forcible_charge_discharge_write", 1},
	47101: {"storage_forcible_charge_discharge_soc", 1},
	47102: {"storage_backup_power_state_of_charge", 1},
	47107: {"storage_unit_1_no", 1},
	47108: {"storage_unit_2_no", 1},
	47200: {"storage_fixed_charging_and_discharging_periods", 41},
	47242: {"storage_power_of_charge_from_grid", 2},
	47244: {"storage_maximum_power_of_charge_from_grid", 2},
	47246: {"storage_forcible_charge_discharge_setting_mode", 1},
	47247: {"storage_forcible_charge_power", 2},
	47249: {"storage_forcible_discharge_power", 2},
	47255: {"storage_tou_charging_and_discharging_periods", 43},
	47299: {"storage_excess_pv_energy_use_in_tou", 1},
	47415: {"active_power_control_mode", 1},
	47416: {"maximum_feed_grid_power_watt", 2},
	47418: {"maximum_feed_grid_power_percent", 1},
	47589: {"remote_charge_discharge_control_mode", 1},
	47590: {"dongle_plant_maximum_charge_from_grid_power", 2},
	47604: {"backup_switch_to_off_grid", 1},
	47605: {"backup_voltage_independent_operation", 1},
	47675: {"default_maximum_feed_in_power", 2},
	47677: {"default_active_power_change_gradient", 2},
	47750: {"storage_unit_1_pack_1_no", 1},
	47751: {"storage_unit_1_pack_2_no", 1},
	47752: {"storage_unit_1_pack_3_no", 1},
	47753: {"storage_unit_2_pack_1_no", 1},
	47754: {"storage_unit_2_pack_2_no", 1},
	47755: {"storage_unit_2_pack_3_no", 1},
	47954: {"storage_capacity_control_mode", 1},
	47955: {"storage_capacity_control_soc_peak_shaving", 1},
	47956: {"storage_capacity_control_periods", 64},
	48020: {"emma", 1},
}

func init() {
	// PV String Data (32016–32063): 24 strings × 2 registers (voltage + current)
	for i := 0; i < 24; i++ {
		base := uint16(32016 + i*2)
		registerDefs[base] = RegisterDef{fmt.Sprintf("pv_%02d_voltage", i+1), 1}
		registerDefs[base+1] = RegisterDef{fmt.Sprintf("pv_%02d_current", i+1), 1}
	}

	// MPPT cumulative yields (32212–32231): 10 MPPTs × 2 registers each
	for i := 0; i < 10; i++ {
		addr := uint16(32212 + i*2)
		registerDefs[addr] = RegisterDef{fmt.Sprintf("cumulative_dc_energy_yield_mppt_%d", i+1), 2}
	}

	// Battery pack detail blocks (38200–38451): 6 packs × 42 registers
	packs := []struct{ unit, pack int }{
		{1, 1}, {1, 2}, {1, 3}, {2, 1}, {2, 2}, {2, 3},
	}
	for i, p := range packs {
		base := uint16(38200 + i*42)
		pfx := fmt.Sprintf("storage_unit_%d_battery_pack_%d", p.unit, p.pack)
		registerDefs[base] = RegisterDef{pfx + "_serial_number", 10}
		registerDefs[base+10] = RegisterDef{pfx + "_firmware_version", 15}
		registerDefs[base+25] = RegisterDef{pfx + "_working_status", 1}
		registerDefs[base+26] = RegisterDef{pfx + "_state_of_capacity", 1}
		registerDefs[base+27] = RegisterDef{pfx + "_charge_discharge_power", 2}
		registerDefs[base+29] = RegisterDef{pfx + "_voltage", 1}
		registerDefs[base+30] = RegisterDef{pfx + "_current", 1}
		registerDefs[base+31] = RegisterDef{pfx + "_total_charge", 2}
		registerDefs[base+33] = RegisterDef{pfx + "_total_discharge", 2}
	}

	// Battery pack temperatures (38452–38463): 6 packs × 2 registers
	for i, p := range packs {
		base := uint16(38452 + i*2)
		pfx := fmt.Sprintf("storage_unit_%d_battery_pack_%d", p.unit, p.pack)
		registerDefs[base] = RegisterDef{pfx + "_max_temperature", 1}
		registerDefs[base+1] = RegisterDef{pfx + "_min_temperature", 1}
	}
}

// RegisterName returns a human-readable name for a register address.
// For multi-register values, continuation addresses return "name +N".
// Returns "" for unknown addresses.
func RegisterName(addr uint16) string {
	if def, ok := registerDefs[addr]; ok {
		return def.Name
	}

	// Check if this address falls within a multi-register value.
	for startAddr, def := range registerDefs {
		if def.Length > 1 && addr > startAddr && addr < startAddr+def.Length {
			return fmt.Sprintf("%s +%d", def.Name, addr-startAddr)
		}
	}

	return ""
}
