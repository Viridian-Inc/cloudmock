export interface DDBAttributeValue {
  S?: string;
  N?: string;
  BOOL?: boolean;
  NULL?: boolean;
  L?: DDBAttributeValue[];
  M?: Record<string, DDBAttributeValue>;
  SS?: string[];
  NS?: string[];
  B?: string;
  BS?: string[];
}

export type DDBItem = Record<string, DDBAttributeValue>;

export interface KeySchemaElement {
  AttributeName: string;
  KeyType: 'HASH' | 'RANGE';
}

export interface AttributeDefinition {
  AttributeName: string;
  AttributeType: 'S' | 'N' | 'B';
}

export interface GSIDescription {
  IndexName: string;
  KeySchema: KeySchemaElement[];
  Projection: { ProjectionType: string; NonKeyAttributes?: string[] };
  IndexStatus: string;
  ItemCount?: number;
}

export interface LSIDescription {
  IndexName: string;
  KeySchema: KeySchemaElement[];
  Projection: { ProjectionType: string; NonKeyAttributes?: string[] };
  ItemCount?: number;
}

export interface TableDescription {
  TableName: string;
  TableArn?: string;
  TableStatus: string;
  KeySchema: KeySchemaElement[];
  AttributeDefinitions: AttributeDefinition[];
  BillingModeSummary?: { BillingMode: string };
  ItemCount?: number;
  TableSizeBytes?: number;
  CreationDateTime?: number;
  GlobalSecondaryIndexes?: GSIDescription[];
  LocalSecondaryIndexes?: LSIDescription[];
  StreamSpecification?: { StreamEnabled: boolean; StreamViewType?: string };
  LatestStreamArn?: string;
}

export interface FilterCondition {
  attribute: string;
  operator: string;
  value: string;
  value2: string; // for "between"
  connector: 'AND' | 'OR';
}

export interface QueryHistoryEntry {
  id: string;
  timestamp: number;
  table: string;
  type: 'query' | 'scan';
  partitionKey: string;
  partitionValue: string;
  sortCondition: string;
  sortValue: string;
  indexName: string;
  resultCount: number;
}

export type DDBType = 'S' | 'N' | 'BOOL' | 'NULL' | 'L' | 'M' | 'SS' | 'NS';

export interface FormAttribute {
  key: string;
  type: DDBType;
  value: string;
  listItems?: FormAttribute[];
  mapItems?: FormAttribute[];
}
