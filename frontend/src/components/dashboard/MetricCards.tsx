import { useState } from 'react';
import { Button, Card, Col, Row, Statistic, Tooltip, Typography } from 'antd';
import { EditOutlined, WarningOutlined } from '@ant-design/icons';
import type { Reading } from '../../types/domain';
import { isAdmin } from '../../utils/auth';
import CalibrationModal from './CalibrationModal';

const labels: Record<string, string> = { temperature: '温度', humidity: '湿度', light: '光照', co2: 'CO₂', soil_moisture: '土壤湿度' };

export default function MetricCards({ readings, onUpdated }: { readings: Reading[]; onUpdated: () => void }) {
  const [editing, setEditing] = useState<Reading | null>(null);
  const admin = isAdmin();
  return (
    <>
      <Row gutter={[16, 16]}>
        {readings.map((row) => {
          const threshold = row.sensor?.threshold;
          const abnormal = Boolean(threshold && (row.value < threshold.minValue || row.value > threshold.maxValue));
          const offset = row.sensor?.calibrationOffset ?? 0;
          return (
            <Col xs={24} sm={12} xl={6} key={row.id}>
              <Card className={abnormal ? 'metric-card abnormal' : 'metric-card'}>
                <Statistic
                  title={labels[row.sensor?.type] ?? '环境参数'}
                  value={row.value}
                  precision={1}
                  suffix={row.sensor?.unit}
                  prefix={abnormal ? <WarningOutlined /> : undefined}
                />
                <small>{abnormal ? '超出配置阈值' : '传感器状态正常'}</small>
                <div className="calibration-line">
                  <Typography.Text type="secondary">
                    校准偏移 {offset > 0 ? `+${offset}` : offset}
                    {offset !== 0 && row.rawValue !== undefined ? ` · 原始值 ${row.rawValue}` : ''}
                  </Typography.Text>
                  {admin && (
                    <Tooltip title="设置校准偏移">
                      <Button type="link" size="small" icon={<EditOutlined />} aria-label="设置校准偏移" onClick={() => setEditing(row)} />
                    </Tooltip>
                  )}
                </div>
              </Card>
            </Col>
          );
        })}
      </Row>
      <CalibrationModal reading={editing} onClose={() => setEditing(null)} onSaved={onUpdated} />
    </>
  );
}
