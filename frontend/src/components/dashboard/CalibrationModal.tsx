import { useEffect, useState } from 'react';
import { InputNumber, Modal, Typography, message } from 'antd';
import type { Reading } from '../../types/domain';
import { updateCalibration } from '../../api/monitoring';
import { apiErrorMessage } from '../../api/client';
import { CALIBRATION_LIMITS } from '../../constants/calibration';

const labels: Record<string, string> = { temperature: '温度', humidity: '湿度', light: '光照', co2: 'CO₂', soil_moisture: '土壤湿度' };

type Props = { reading: Reading | null; onClose: () => void; onSaved: () => void };

export default function CalibrationModal({ reading, onClose, onSaved }: Props) {
  const [offset, setOffset] = useState(0);
  const [saving, setSaving] = useState(false);
  const sensor = reading?.sensor;
  const limit = sensor ? CALIBRATION_LIMITS[sensor.type] : undefined;

  useEffect(() => {
    setOffset(sensor?.calibrationOffset ?? 0);
  }, [sensor]);

  const save = async () => {
    if (!sensor || !limit) return;
    setSaving(true);
    try {
      await updateCalibration(sensor.id, offset);
      message.success(offset === 0 ? '校准偏移已清零，恢复显示原始读数' : '校准偏移已保存');
      onSaved();
      onClose();
    } catch (error) {
      message.error(apiErrorMessage(error, '校准偏移保存失败，原有偏移保持不变'));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      open={Boolean(reading)}
      title={`校准 ${sensor?.name ?? labels[sensor?.type ?? ''] ?? '传感器'}`}
      okText="保存"
      cancelText="取消"
      confirmLoading={saving}
      onOk={save}
      onCancel={onClose}
      destroyOnClose
    >
      {sensor && limit && (
        <Typography.Paragraph type="secondary">
          保存后，最新读数、历史趋势、阈值报警与环境报告均使用校准值；原始读数仍会保留，偏移填 0 即可恢复原始值。
        </Typography.Paragraph>
      )}
      {sensor && limit && (
        <div className="calibration-form">
          <Typography.Text>校准偏移（{sensor.unit}）</Typography.Text>
          <InputNumber
            value={offset}
            min={limit.min}
            max={limit.max}
            step={sensor.type === 'light' || sensor.type === 'co2' ? 1 : 0.1}
            onChange={(value) => setOffset(value ?? 0)}
            aria-label="校准偏移"
          />
          <Typography.Text type="secondary">
            允许范围 {limit.min} ~ {limit.max} {sensor.unit}
            {reading && reading.rawValue !== undefined ? ` · 当前原始读数 ${reading.rawValue}` : ''}
          </Typography.Text>
        </div>
      )}
    </Modal>
  );
}
