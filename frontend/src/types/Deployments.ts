export interface DeploymentsModel {
  deployments: Deployment[];
  // aggregations: AggregationModel[];
  // charts: ChartModel[];
  // externalLinks: ExternalLink[];
  // rows: number;
  // title: string;
}

export type Deployment = {
  added?: Commit[];
  comparison_url?: string;
  created_at: Date;
  id: number;
  removed?: Commit[];
  sha: string;
  updated_at: Date;
};

export type Commit = {
  sha: string;
  title: string;
  url: string;
};
