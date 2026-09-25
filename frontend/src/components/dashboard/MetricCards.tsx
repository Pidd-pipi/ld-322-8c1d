import { useState } from 'react';
import { Card, Col, InputNumber, Row, Statistic, Tooltip, Typography, message } from 'antd';
import { WarningOutlined } from '@ant-design/icons';
import type { Reading } from '../../types/domain';
import { maxCalibrationOffset, saveCalibration } from '../../api/calibration';

const labels: Record<string, string> = {
  temperature: '温度',
  humidity: '湿度',
  light: '光照',
  co2: 'CO₂',
  soil_moisture: '土壤湿度',
};

type Props = {
  readings: Reading[];
  onSaved: () => void;
};

function MetricCard({ reading, onSaved }: { reading: Reading; onSaved: Props['onSaved'] }) {
  const sensor = reading.sensor;
  const [offset, setOffset] = useState<number | null>(sensor?.calibrationOffset ?? 0);
  const [saving, setSaving] = useState(false);

  const limit = maxCalibrationOffset[sensor?.type] ?? 0;
  const calibrated = reading.calibratedValue ?? reading.value;
  const raw = reading.value;
  const hasOffset = (sensor?.calibrationOffset ?? 0) !== 0;
  const abnormal = Boolean(
    sensor?.threshold && (calibrated < sensor.threshold.minValue || calibrated > sensor.threshold.maxValue),
  );

  const save = async () => {
    if (!sensor) {
      message.error('传感器信息缺失，无法保存校准偏移');
      return;
    }
    const value = offset ?? 0;
    if (Number.isNaN(value)) {
      message.error('请输入有效的偏移数值');
      return;
    }
    if (value < -limit || value > limit) {
      message.error(`校准偏移超出允许范围：${labels[sensor.type] ?? sensor.type} 只能在 -${limit} 到 ${limit} 之间`);
      return;
    }
    setSaving(true);
    try {
      await saveCalibration(sensor.id, value);
      message.success(value === 0 ? '偏移已清零，恢复原始读数' : '校准偏移已保存');
      onSaved();
    } catch (err) {
      message.error(err instanceof Error ? err.message : '校准偏移保存失败，原数据未改动');
      setOffset(sensor.calibrationOffset ?? 0);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Card className={abnormal ? 'metric-card abnormal' : 'metric-card'}>
      <Statistic
        title={labels[sensor?.type] ?? '环境参数'}
        value={calibrated}
        precision={1}
        suffix={sensor?.unit}
        prefix={abnormal ? <WarningOutlined /> : undefined}
      />
      <div className="metric-meta">
        <small>{abnormal ? '校准值超出配置阈值' : '传感器状态正常'}</small>
        {hasOffset && (
          <Tooltip title="原始读数始终保留，偏移清零即恢复原值">
            <Typography.Text type="secondary" className="raw-value">
              原始 {raw.toFixed(1)}
              {sensor?.unit} · 偏移 {sensor.calibrationOffset > 0 ? '+' : ''}
              {sensor.calibrationOffset}
            </Typography.Text>
          </Tooltip>
        )}
      </div>
      <div className="calibration-row" onClick={(event) => event.stopPropagation()}>
        <span className="calibration-label">校准偏移</span>
        <InputNumber
          aria-label="校准偏移"
          size="small"
          value={offset}
          min={-limit}
          max={limit}
          step={sensor?.type === 'light' ? 100 : 0.1}
          style={{ width: 104 }}
          onChange={setOffset}
          onPressEnter={() => void save()}
        />
        <button type="button" className="calibration-save" disabled={saving} onClick={() => void save()}>
          {saving ? '保存中' : '保存'}
        </button>
        <small className="calibration-range">±{limit}</small>
      </div>
    </Card>
  );
}

export default function MetricCards({ readings, onSaved }: Props) {
  return (
    <Row gutter={[16, 16]}>
      {readings.map((row) => (
        <Col xs={24} sm={12} xl={6} key={row.id}>
          <MetricCard reading={row} onSaved={onSaved} />
        </Col>
      ))}
    </Row>
  );
}
