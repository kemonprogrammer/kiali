export interface DeploymentsModel {
  deployments: Deployment[];
  total: number;
}

export type Deployment = {
  added?: Commit[];
  comparison_url?: string;
  created_at: Date;
  id: number;
  removed?: Commit[];
  sha: string;
  succeeded_at: Date;
  updated_at: Date;
};

export type Commit = {
  sha: string;
  title: string;
  url: string;
};
