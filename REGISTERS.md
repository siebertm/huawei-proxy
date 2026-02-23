# Huawei SUN2000 Modbus Register Map & Proxy Strategy

## Overview

This document maps all Modbus holding registers used by the
[huawei_solar](https://github.com/wlcrs/huawei_solar) Home Assistant
integration (v2.5.0) and defines the caching proxy strategy.

All registers are **holding registers** (Modbus function code **0x03**).

---

## How the HA Integration Reads Registers

The `huawei_solar` Python library (`batch_update`) reads registers like this:

1. Collects all register names requested by enabled entities
2. Sorts by address
3. Greedily batches consecutive registers together, constrained by:
   - **Max 64 registers** total span per batch read
   - **Max 16 register gap** between adjacent registers in a batch
4. Sends one `read_holding_registers` call per batch

Each batch becomes **one Modbus TCP read request** to the inverter.
The inverter can only handle one connection and needs time between
requests — hence the need for this caching proxy.

---

## Complete SUN2000 Register Map

Extracted from `huawei_solar.registers.REGISTERS`. Only `TargetDevice.SUN2000`
registers are listed (EMMA, SCharger, SDongle, SmartLogger omitted — enable the
relevant groups in config if you use those devices).

### Device Information (30000–31210) — SLOW POLL

These rarely change (model, serial, firmware). Read every 5 minutes.

| Address | Len | Name | Writable |
|---------|-----|------|----------|
| 30000 | 15 | model_name | - |
| 30015 | 10 | serial_number | - |
| 30025 | 10 | pn | - |
| 30035 | 15 | firmware_version | - |
| 30050 | 15 | software_version | - |
| 30068 | 2 | protocol_version_modbus | - |
| 30070 | 1 | model_id | - |
| 30071 | 1 | nb_pv_strings | - |
| 30072 | 1 | nb_mpp_tracks | - |
| 30073 | 2 | rated_power | - |
| 30075 | 2 | P_max | - |
| 30077 | 2 | S_max | - |
| 30079 | 2 | Q_max_out | - |
| 30081 | 2 | Q_max_in | - |
| 30083 | 2 | P_max_real | - |
| 30085 | 2 | S_max_real | - |
| 30105 | 2 | product_sales_area | - |
| 30107 | 1 | product_software_number | - |
| 30108 | 1 | product_software_version_number | - |
| 30109 | 1 | grid_standard_code_protocol_version | - |
| 30110 | 1 | unique_id_of_the_software | - |
| 30111 | 1 | number_of_packages_to_be_upgraded | - |
| 30206 | 1 | hardware_functional_unit_conf_id | - |
| 30207 | 2 | subdevice_support_flag | - |
| 30209 | 2 | subdevice_in_position_flag | - |
| 30211 | 2 | feature_mask_1 | - |
| 30213 | 2 | feature_mask_2 | - |
| 30215 | 2 | feature_mask_3 | - |
| 30217 | 2 | feature_mask_4 | - |
| 30366 | 2 | realtime_max_active_capability | - |
| 30368 | 2 | realtime_max_inductive_reactive_capacity | - |
| 30561 | 15 | offering_name_of_southbound_device_1 | - |
| 30576 | 15 | offering_name_of_southbound_device_2 | - |
| 30591 | 15 | offering_name_of_southbound_device_3 | - |
| 31000 | 15 | hardware_version | - |
| 31015 | 10 | monitoring_board_sn | - |
| 31025 | 15 | monitoring_software_version | - |
| 31040 | 15 | master_dsp_version | - |
| 31055 | 15 | slave_dsp_version | - |
| 31070 | 15 | cpld_version | - |
| 31085 | 15 | afci_version | - |
| 31100 | 15 | builtin_pid_version | - |
| 31115 | 15 | dc_mbus_version | - |
| 31130 | 15 | el_module_version | - |
| 31145 | 15 | afci_2_version | - |
| 31200 | 10 | regkey | - |

### Inverter State & Alarms (32000–32010) — FAST POLL

| Address | Len | Name | Writable |
|---------|-----|------|----------|
| 32000 | 1 | state_1 | - |
| 32002 | 1 | state_2 | - |
| 32003 | 2 | state_3 | - |
| 32008 | 1 | alarm_1 | - |
| 32009 | 1 | alarm_2 | - |
| 32010 | 1 | alarm_3 | - |

### PV String Data (32016–32063) — FAST POLL

48 registers: voltage + current for up to 24 PV strings.

| Address | Len | Name |
|---------|-----|------|
| 32016 | 1 | pv_01_voltage |
| 32017 | 1 | pv_01_current |
| 32018 | 1 | pv_02_voltage |
| ... | | ... (2 regs per string) |
| 32062 | 1 | pv_24_voltage |
| 32063 | 1 | pv_24_current |

### Grid Output & Power (32064–32097) — FAST POLL

| Address | Len | Name | Writable |
|---------|-----|------|----------|
| 32064 | 2 | input_power | - |
| 32066 | 1 | line_voltage_A_B / grid_voltage | - |
| 32067 | 1 | line_voltage_B_C | - |
| 32068 | 1 | line_voltage_C_A | - |
| 32069 | 1 | phase_A_voltage | - |
| 32070 | 1 | phase_B_voltage | - |
| 32071 | 1 | phase_C_voltage | - |
| 32072 | 2 | phase_A_current / grid_current | - |
| 32074 | 2 | phase_B_current | - |
| 32076 | 2 | phase_C_current | - |
| 32078 | 2 | day_active_power_peak | - |
| 32080 | 2 | active_power | - |
| 32082 | 2 | reactive_power | - |
| 32084 | 1 | power_factor | - |
| 32085 | 1 | grid_frequency | - |
| 32086 | 1 | efficiency | - |
| 32087 | 1 | internal_temperature | - |
| 32088 | 1 | insulation_resistance | - |
| 32089 | 1 | device_status | - |
| 32090 | 1 | fault_code | - |
| 32091 | 2 | startup_time | - |
| 32093 | 2 | shutdown_time | - |
| 32095 | 2 | active_power_fast | - |

### Energy Yield (32106–32120) — FAST POLL

| Address | Len | Name |
|---------|-----|------|
| 32106 | 2 | accumulated_yield_energy |
| 32108 | 2 | total_dc_input_power |
| 32110 | 2 | current_electricity_generation_statistics_time |
| 32112 | 2 | hourly_yield_energy |
| 32114 | 2 | daily_yield_energy |
| 32116 | 2 | monthly_yield_energy |
| 32118 | 2 | yearly_yield_energy |

### Diagnostic (32172–32231) — SLOW POLL

| Address | Len | Name |
|---------|-----|------|
| 32172 | 2 | latest_active_alarm_sn |
| 32174 | 2 | latest_historical_alarm_sn |
| 32176 | 1 | total_bus_voltage |
| 32177 | 1 | max_pv_voltage |
| 32178 | 1 | min_pv_voltage |
| 32179 | 1 | average_pv_negative_voltage_to_ground |
| 32180 | 1 | min_pv_negative_voltage_to_ground |
| 32181 | 1 | max_pv_negative_voltage_to_ground |
| 32182 | 1 | inverter_to_pe_voltage_tolerance |
| 32183 | 1 | iso_feature_information |
| 32190 | 1 | builtin_pid_running_status |
| 32191 | 1 | pv_negative_voltage_to_ground |
| 32212–32231 | 2 each | cumulative_dc_energy_yield_mppt1–10 |

### Component Health (35000–35044) — SLOW POLL

| Address | Len | Name |
|---------|-----|------|
| 35000 | 2 | capbank_running_time |
| 35002 | 2 | internal_fan_1_running_time |
| 35021 | 1 | inv_module_a_temp |
| 35022 | 1 | inv_module_b_temp |
| 35023 | 1 | inv_module_c_temp |
| 35024 | 1 | anti_reverse_module_1_temp |
| 35025 | 1 | output_board_relay_ambient_temp_max |
| 35027 | 1 | anti_reverse_module_2_temp |
| 35028 | 1 | dc_terminal_1_2_max_temp |
| 35029 | 1 | ac_terminal_1_2_3_max_temp |
| 35038 | 1 | phase_a_dc_component_dci |
| 35039 | 1 | phase_b_dc_component_dci |
| 35040 | 1 | phase_c_dc_component_dci |
| 35041 | 1 | leakage_current_rcd |
| 35042 | 1 | positive_bus_voltage |
| 35043 | 1 | negative_bus_voltage |
| 35044 | 1 | bus_negative_voltage_to_ground |

### Storage Unit 1 (37000–37068) — FAST POLL

| Address | Len | Name |
|---------|-----|------|
| 37000 | 1 | storage_unit_1_running_status |
| 37001 | 2 | storage_unit_1_charge_discharge_power |
| 37003 | 1 | storage_unit_1_bus_voltage |
| 37004 | 1 | storage_unit_1_state_of_capacity |
| 37006 | 1 | storage_unit_1_working_mode_b |
| 37007 | 2 | storage_unit_1_rated_charge_power |
| 37009 | 2 | storage_unit_1_rated_discharge_power |
| 37014 | 1 | storage_unit_1_fault_id |
| 37015 | 2 | storage_unit_1_current_day_charge_capacity |
| 37017 | 2 | storage_unit_1_current_day_discharge_capacity |
| 37021 | 1 | storage_unit_1_bus_current |
| 37022 | 1 | storage_unit_1_battery_temperature |
| 37025 | 1 | storage_unit_1_remaining_charge_dis_charge_time |
| 37026 | 10 | storage_unit_1_dcdc_version |
| 37036 | 10 | storage_unit_1_bms_version |
| 37046 | 2 | storage_maximum_charge_power |
| 37048 | 2 | storage_maximum_discharge_power |
| 37052 | 10 | storage_unit_1_serial_number |
| 37066 | 2 | storage_unit_1_total_charge |
| 37068 | 2 | storage_unit_1_total_discharge |

### Power Meter (37100–37139) — FAST POLL

| Address | Len | Name |
|---------|-----|------|
| 37100 | 1 | meter_status |
| 37101 | 2 | grid_A_voltage |
| 37103 | 2 | grid_B_voltage |
| 37105 | 2 | grid_C_voltage |
| 37107 | 2 | active_grid_A_current |
| 37109 | 2 | active_grid_B_current |
| 37111 | 2 | active_grid_C_current |
| 37113 | 2 | power_meter_active_power |
| 37115 | 2 | power_meter_reactive_power |
| 37117 | 1 | active_grid_power_factor |
| 37118 | 1 | active_grid_frequency |
| 37119 | 2 | grid_exported_energy |
| 37121 | 2 | grid_accumulated_energy |
| 37123 | 2 | grid_accumulated_reactive_power |
| 37125 | 1 | meter_type |
| 37126 | 2 | active_grid_A_B_voltage |
| 37128 | 2 | active_grid_B_C_voltage |
| 37130 | 2 | active_grid_C_A_voltage |
| 37132 | 2 | active_grid_A_power |
| 37134 | 2 | active_grid_B_power |
| 37136 | 2 | active_grid_C_power |
| 37138 | 1 | meter_type_check |

### Optimizers (37200–37201) — SLOW POLL

| Address | Len | Name |
|---------|-----|------|
| 37200 | 1 | nb_optimizers |
| 37201 | 1 | nb_online_optimizers |

### Storage Unit 2 (37700–37829) — FAST POLL

| Address | Len | Name |
|---------|-----|------|
| 37700 | 10 | storage_unit_2_serial_number |
| 37738 | 1 | storage_unit_2_state_of_capacity |
| 37741 | 1 | storage_unit_2_running_status |
| 37743 | 2 | storage_unit_2_charge_discharge_power |
| 37746 | 2 | storage_unit_2_current_day_charge_capacity |
| 37748 | 2 | storage_unit_2_current_day_discharge_capacity |
| 37750 | 1 | storage_unit_2_bus_voltage |
| 37751 | 1 | storage_unit_2_bus_current |
| 37752 | 1 | storage_unit_2_battery_temperature |
| 37753 | 2 | storage_unit_2_total_charge |
| 37755 | 2 | storage_unit_2_total_discharge |
| 37799 | 15 | storage_unit_2_software_version |
| 37814 | 15 | storage_unit_1_software_version |

### Storage Aggregated (37758–37788) — FAST POLL

| Address | Len | Name |
|---------|-----|------|
| 37758 | 2 | storage_rated_capacity |
| 37760 | 1 | storage_state_of_capacity |
| 37762 | 1 | storage_running_status |
| 37763 | 1 | storage_bus_voltage |
| 37764 | 1 | storage_bus_current |
| 37765 | 2 | storage_charge_discharge_power |
| 37780 | 2 | storage_total_charge |
| 37782 | 2 | storage_total_discharge |
| 37784 | 2 | storage_current_day_charge_capacity |
| 37786 | 2 | storage_current_day_discharge_capacity |

### Battery Pack SOH (37920–37928) — SLOW POLL

| Address | Len | Name |
|---------|-----|------|
| 37920–37925 | 1 each | storage_unit_{1,2}_battery_pack_{1,2,3}_soh_calibration_status |
| 37926 | 1 | storage_unit_soh_calibration_status |
| 37927 | 1 | storage_unit_soh_calibration_release_lower_limit_of_soc |

### Battery Pack Details (38200–38463) — SLOW POLL

6 battery packs (unit 1 packs 1-3, unit 2 packs 1-3), each ~42 registers:
serial number, firmware version, working status, SOC, charge/discharge power,
voltage, current, total charge/discharge. Plus temperature registers at
38452–38463.

| Block | Address Range | Description |
|-------|---------------|-------------|
| Unit 1, Pack 1 | 38200–38241 | SN, FW, status, SOC, power, V, I, totals |
| Unit 1, Pack 2 | 38242–38283 | Same structure |
| Unit 1, Pack 3 | 38284–38325 | Same structure |
| Unit 2, Pack 1 | 38326–38367 | Same structure |
| Unit 2, Pack 2 | 38368–38409 | Same structure |
| Unit 2, Pack 3 | 38410–38451 | Same structure |
| Temperatures | 38452–38463 | Min/max temps for all 6 packs |

### System Time (40000–40001) — READ ONLY

| Address | Len | Name |
|---------|-----|------|
| 40000 | 2 | system_time |

### Inverter Control (40037–40201) — SLOW POLL (config)

| Address | Len | Name | Writable |
|---------|-----|------|----------|
| 40037 | 1 | q_u_characteristic_curve_model | W |
| 40038 | 1 | q_u_scheduling_trigger_power_percentage | W |
| 40122 | 1 | power_factor_2 | W |
| 40123 | 1 | reactive_power_compensation | W |
| 40124 | 1 | reactive_power_adjustment_time | W |
| 40125 | 1 | active_power_percentage_derating | W |
| 40126 | 2 | active_power_fixed_value_derating | W |
| 40128 | 1 | reactive_power_compensation_at_night | W |
| 40129 | 2 | fixed_reactive_power_at_night | W |
| 40196 | 1 | characteristic_curve_reactive_power_adjustment_time | W |
| 40197 | 1 | percent_apparent_power | W |
| 40198 | 1 | q_u_scheduling_exit_power_percentage | W |
| 40200 | 1 | startup | W |
| 40201 | 1 | shutdown | W |

### Grid & MPPT Config (42000–42057) — SLOW POLL

| Address | Len | Name | Writable |
|---------|-----|------|----------|
| 42000 | 1 | grid_code | - |
| 42054 | 1 | mppt_multimodal_scanning | W |
| 42055 | 1 | mppt_scanning_interval | W |
| 42056 | 2 | mppt_predicted_power | - |
| 42900 | 1 | daylight_saving_time | W |
| 43006 | 1 | time_zone | W |

### Storage Configuration (47000–48020) — SLOW POLL (config)

| Address | Len | Name | Writable |
|---------|-----|------|----------|
| 47000 | 1 | storage_unit_1_product_model | - |
| 47004 | 1 | storage_working_mode_a | - |
| 47027 | 1 | storage_time_of_use_price | - |
| 47028 | 41 | storage_time_of_use_price_periods | W |
| 47069 | 2 | storage_lcoe | - |
| 47075 | 2 | storage_maximum_charging_power | W |
| 47077 | 2 | storage_maximum_discharging_power | W |
| 47079 | 2 | storage_power_limit_grid_tied_point | - |
| 47081 | 1 | storage_charging_cutoff_capacity | W |
| 47082 | 1 | storage_discharging_cutoff_capacity | W |
| 47083 | 1 | storage_forced_charging_and_discharging_period | W |
| 47084 | 2 | storage_forced_charging_and_discharging_power | - |
| 47086 | 1 | storage_working_mode_settings | W |
| 47087 | 1 | storage_charge_from_grid_function | W |
| 47088 | 1 | storage_grid_charge_cutoff_state_of_charge | W |
| 47089 | 1 | storage_unit_2_product_model | - |
| 47100 | 1 | forcible_charge_discharge_write | W |
| 47101 | 1 | storage_forcible_charge_discharge_soc | W |
| 47102 | 1 | storage_backup_power_state_of_charge | W |
| 47107 | 1 | storage_unit_1_no | - |
| 47108 | 1 | storage_unit_2_no | - |
| 47200 | 41 | storage_fixed_charging_and_discharging_periods | W |
| 47242 | 2 | storage_power_of_charge_from_grid | W |
| 47244 | 2 | storage_maximum_power_of_charge_from_grid | W |
| 47246 | 1 | storage_forcible_charge_discharge_setting_mode | W |
| 47247 | 2 | storage_forcible_charge_power | W |
| 47249 | 2 | storage_forcible_discharge_power | W |
| 47255 | 43 | storage_tou_charging_and_discharging_periods | W |
| 47299 | 1 | storage_excess_pv_energy_use_in_tou | W |
| 47415 | 1 | active_power_control_mode | W |
| 47416 | 2 | maximum_feed_grid_power_watt | W |
| 47418 | 1 | maximum_feed_grid_power_percent | W |
| 47589 | 1 | remote_charge_discharge_control_mode | W |
| 47590 | 2 | dongle_plant_maximum_charge_from_grid_power | W |
| 47604 | 1 | backup_switch_to_off_grid | W |
| 47605 | 1 | backup_voltage_independent_operation | W |
| 47675 | 2 | default_maximum_feed_in_power | W |
| 47677 | 2 | default_active_power_change_gradient | - |
| 47750–47755 | 1 each | storage_unit_{1,2}_pack_{1,2,3}_no | - |
| 47954 | 1 | storage_capacity_control_mode | W |
| 47955 | 1 | storage_capacity_control_soc_peak_shaving | W |
| 47956 | 64 | storage_capacity_control_periods | W |
| 48020 | 1 | emma | W |

---

## Proxy Strategy

### Architecture

```
┌──────────────┐         ┌──────────────────────┐         ┌───────────┐
│  HA instance │ ◄─────► │   huawei-solar-proxy  │ ◄─────► │  SUN2000  │
│  (huawei_    │  fast   │                      │  500ms   │  Inverter │
│   solar)     │  TCP    │  ┌──────────────┐    │  gaps    │  :502     │
│              │         │  │ register     │    │          │           │
│              │         │  │ cache        │    │          │           │
│              │         │  └──────────────┘    │          │           │
└──────────────┘         └──────────────────────┘         └───────────┘
```

### The Problem

The Huawei SUN2000 inverter:
- Supports only **one Modbus TCP connection** at a time
- Needs **~500ms minimum gap** between read calls or it becomes unresponsive
- The HA integration sends **many batch reads** per update cycle (every 30s)
- Each batch may contain **multiple Modbus reads** (up to 64 regs each)

### The Solution

A Go proxy that:

1. **Continuously reads** all configured register groups from the inverter,
   respecting the 500ms inter-read pause
2. **Caches** all register values in memory
3. **Serves** Modbus TCP requests from HA instantly from cache
4. **Forwards writes** to the inverter (for config changes, forcible charge, etc.)
5. **Forwards unknown reads** directly to the inverter on cache miss (configurable)

### Two-Tier Polling

**Fast poll** (continuous cycle, ~8-12 seconds per cycle):
- Inverter state, alarms (32000–32010)
- PV string data (32016–32063)
- Grid output & power (32064–32097)
- Energy yield (32106–32120)
- Storage unit 1 status (37000–37070)
- Power meter (37100–37139)
- Storage unit 2 + aggregated (37738–37788)

**Slow poll** (every 5 minutes):
- Device info (30000–31210)
- Diagnostics (32172–32231)
- Component health (35000–35044)
- Optimizer info (37200–37201)
- Battery pack details (37920–38463)
- All configuration registers (40xxx, 42xxx, 47xxx)

### Timing Math

Fast cycle (7 groups):
- 7 reads × 500ms gap = **~3.5 seconds** per fast cycle
- Operational data refreshes every ~3.5s

Slow cycle (adds ~20 groups):
- ~20 reads × 500ms = ~10 extra seconds
- Runs every 5 minutes, so fast cycle is briefly paused

### Register Group Sizing

Each read is constrained by:
- **Modbus limit**: max 125 registers per read
- **Inverter behavior**: reading over gaps (non-existent addresses) within a
  valid range generally returns 0 — the inverter doesn't error

Groups are sized to cover contiguous or near-contiguous address blocks
within the 125-register limit.

### Write Handling

When HA writes a register (e.g., forcible charge/discharge):
1. Proxy receives the Modbus write request
2. Acquires the inverter mutex (waits for 500ms gap)
3. Forwards the write to the inverter
4. Updates the cache with the written value
5. Returns the response to HA

### Cache Miss Handling

When `forward_unknown_reads: true` (default):
- Registers not in any configured group can still be read
- The proxy forwards the request to the inverter (respecting 500ms gap)
- The response is cached for future requests
- This handles HA's device detection probing at startup

When `forward_unknown_reads: false`:
- Cache misses return Modbus exception 0x02 (Illegal Data Address)
- All register groups must be pre-configured

### Startup Sequence

1. Load YAML config
2. Connect to inverter
3. **Initial scan**: read ALL groups (both fast and slow) once
4. Start the reader loop goroutine
5. Start the Modbus TCP server
6. HA connects and gets instant responses

---

## Recommended Register Groups (Default Config)

These are the default register groups for the proxy config. Remove groups
you don't need (e.g., storage groups if you have no battery).

### Fast Groups

| Name | Address | Count | Notes |
|------|---------|-------|-------|
| inverter_state | 32000 | 11 | state_1/2/3 + alarm_1/2/3 |
| pv_strings | 32016 | 48 | PV 1-24 voltage+current |
| grid_power | 32064 | 33 | Input power through shutdown_time |
| energy_yield | 32106 | 14 | Accumulated + daily/monthly/yearly |
| storage_unit_1 | 37000 | 70 | Storage unit 1 full status |
| meter | 37100 | 39 | Power meter all data |
| storage_unit_2_status | 37738 | 50 | Storage unit 2 + aggregated |

### Slow Groups

| Name | Address | Count | Notes |
|------|---------|-------|-------|
| device_info_1 | 30000 | 65 | Model, SN, FW, software |
| device_info_2 | 30068 | 44 | Model ID through product info |
| device_features | 30206 | 14 | HW config + feature masks |
| hw_versions_1 | 31000 | 70 | HW version through slave_dsp |
| hw_versions_2 | 31070 | 90 | CPLD through afci_2 |
| regkey | 31200 | 10 | Registration key |
| diagnostics_1 | 32172 | 20 | Alarm SNs + bus/PV voltages |
| diagnostics_2 | 32212 | 20 | MPPT cumulative yields |
| component_health_1 | 35000 | 4 | Capbank + fan running time |
| component_health_2 | 35021 | 24 | Module temps + DCI + bus V |
| optimizer_info | 37200 | 2 | Optimizer count |
| storage_versions | 37799 | 30 | SW versions for units 1+2 |
| battery_soh | 37920 | 8 | SOH calibration statuses |
| battery_packs_1 | 38200 | 84 | Unit 1 packs 1-2 |
| battery_packs_2 | 38284 | 84 | Unit 1 pack 3 + unit 2 pack 1 |
| battery_packs_3 | 38368 | 84 | Unit 2 packs 2-3 |
| battery_temps | 38452 | 12 | All pack temperatures |
| system_time | 40000 | 2 | Current time |
| inverter_config_1 | 40122 | 10 | Power factor + derating |
| inverter_config_2 | 40196 | 6 | Reactive power + startup/shutdown |
| grid_mppt_config | 42054 | 4 | MPPT scanning settings |
| storage_config_1 | 47000 | 90 | Product model through unit 2 no |
| storage_config_2 | 47200 | 100 | Fixed periods + TOU + forcible |
| power_control | 47415 | 4 | Active power control mode |
| storage_config_3 | 47589 | 90 | Remote control through pack nos |
| capacity_control | 47954 | 67 | Capacity control + periods |
