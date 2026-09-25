import client from './client';
import type { ApiResponse, Sensor, SensorType } from '../types/domain';

// 与后端 constants.MaxCalibrationOffset 保持一致，用于前端输入约束与提示。
export const maxCalibrationOffset: Record<SensorType, number> = {
  temperature: 5,
  humidity: 10,
  light: 5000,
  co2: 200,
  soil_moisture: 10,
};

export const saveCalibration = async (sensorId: number, offset: number) =>
  (await client.put<ApiResponse<Sensor>>(`/sensors/${sensorId}/calibration`, { offset })).data.data;
