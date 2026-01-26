import { LabelDisplayName, PromLabel, SingleLabelValues } from 'types/Metrics';

export type Quantiles = '0.5' | '0.95' | '0.99' | '0.999';
export const allQuantiles: Quantiles[] = ['0.5', '0.95', '0.99', '0.999'];

export type LabelSettings = {
  checked: boolean;
  defaultValue: boolean;
  displayName: LabelDisplayName;
  singleSelection: boolean;
  values: SingleLabelValues;
};
export type LabelsSettings = Map<PromLabel, LabelSettings>;

export interface MetricsSettings {
  labelsSettings: LabelsSettings;
  showAverage: boolean;
  showDeployments: boolean;
  showQuantiles: Quantiles[];
  showSpans: boolean;
  showTrendlines: boolean;
  spanLimit: number;
}
