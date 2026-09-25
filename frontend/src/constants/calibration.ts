// 与后端 constants.CalibrationLimits 保持一致；后端仍会再次校验，此处仅用于表单提示与输入限制。
export const CALIBRATION_LIMITS: Record<string, { min: number; max: number }> = {
  temperature: { min: -10, max: 10 },
  humidity: { min: -20, max: 20 },
  light: { min: -5000, max: 5000 },
  co2: { min: -500, max: 500 },
  soil_moisture: { min: -20, max: 20 },
};
