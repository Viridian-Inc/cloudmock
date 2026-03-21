import { useState, useMemo } from 'preact/hooks';
import { Modal } from '../../components/Modal';
import { CopyIcon } from '../../components/Icons';
import { copyToClipboard } from '../../utils';
import { DDBItem, TableDescription, FilterCondition } from './types';

interface CodeGeneratorProps {
  tableName: string;
  tableDesc: TableDescription;
  mode: 'query' | 'scan';
  pkValue: string;
  skOp: string;
  skValue: string;
  skValue2: string;
  indexName: string;
  filters: FilterCondition[];
  limit: string;
  onClose: () => void;
  showToast: (msg: string) => void;
}

type Language = 'javascript' | 'python' | 'go';

export function CodeGenerator({
  tableName, tableDesc, mode, pkValue, skOp, skValue, skValue2,
  indexName, filters, limit, onClose, showToast,
}: CodeGeneratorProps) {
  const [lang, setLang] = useState<Language>('javascript');

  const indexes = useMemo(() => {
    const list: { name: string; pk: string; sk: string }[] = [
      {
        name: '',
        pk: tableDesc.KeySchema.find(k => k.KeyType === 'HASH')?.AttributeName || '',
        sk: tableDesc.KeySchema.find(k => k.KeyType === 'RANGE')?.AttributeName || '',
      },
    ];
    if (tableDesc.GlobalSecondaryIndexes) {
      for (const gsi of tableDesc.GlobalSecondaryIndexes) {
        list.push({
          name: gsi.IndexName,
          pk: gsi.KeySchema.find(k => k.KeyType === 'HASH')?.AttributeName || '',
          sk: gsi.KeySchema.find(k => k.KeyType === 'RANGE')?.AttributeName || '',
        });
      }
    }
    return list;
  }, [tableDesc]);

  const activeIndex = indexes.find(i => i.name === indexName) || indexes[0];
  const activePkType = tableDesc.AttributeDefinitions.find(a => a.AttributeName === activeIndex.pk)?.AttributeType || 'S';
  const activeSkType = tableDesc.AttributeDefinitions.find(a => a.AttributeName === activeIndex.sk)?.AttributeType || 'S';

  function generateJS(): string {
    if (mode === 'scan') {
      let code = `const { DynamoDBClient, ScanCommand } = require("@aws-sdk/client-dynamodb");\n`;
      code += `const client = new DynamoDBClient({ region: "us-east-1" });\n\n`;
      const params: string[] = [`  TableName: "${tableName}"`,];
      if (limit) params.push(`  Limit: ${limit}`);
      const filterParts = buildJSFilter(filters);
      if (filterParts) {
        params.push(`  FilterExpression: "${filterParts.expr}"`);
        params.push(`  ExpressionAttributeNames: ${filterParts.names}`);
        params.push(`  ExpressionAttributeValues: ${filterParts.values}`);
      }
      code += `const result = await client.send(new ScanCommand({\n${params.join(",\n")}\n}));\n`;
      code += `console.log(result.Items);`;
      return code;
    }

    // Query mode
    let code = `const { DynamoDBClient, QueryCommand } = require("@aws-sdk/client-dynamodb");\n`;
    code += `const client = new DynamoDBClient({ region: "us-east-1" });\n\n`;
    const params: string[] = [`  TableName: "${tableName}"`,];
    if (indexName) params.push(`  IndexName: "${indexName}"`);

    // Build key condition
    let keyExpr = `${activeIndex.pk} = :pk`;
    const exprValues: Record<string, string> = {};
    exprValues[':pk'] = activePkType === 'N' ? `{ "N": "${pkValue}" }` : `{ "S": "${pkValue}" }`;

    if (activeIndex.sk && skValue) {
      if (skOp === 'begins_with') {
        keyExpr += ` AND begins_with(${activeIndex.sk}, :sk)`;
      } else if (skOp === 'between') {
        keyExpr += ` AND ${activeIndex.sk} BETWEEN :sk AND :sk2`;
        exprValues[':sk2'] = activeSkType === 'N' ? `{ "N": "${skValue2}" }` : `{ "S": "${skValue2}" }`;
      } else {
        keyExpr += ` AND ${activeIndex.sk} ${skOp} :sk`;
      }
      exprValues[':sk'] = activeSkType === 'N' ? `{ "N": "${skValue}" }` : `{ "S": "${skValue}" }`;
    }

    params.push(`  KeyConditionExpression: "${keyExpr}"`);
    const valLines = Object.entries(exprValues).map(([k, v]) => `    "${k}": ${v}`).join(",\n");
    params.push(`  ExpressionAttributeValues: {\n${valLines}\n  }`);
    if (limit) params.push(`  Limit: ${limit}`);

    code += `const result = await client.send(new QueryCommand({\n${params.join(",\n")}\n}));\n`;
    code += `console.log(result.Items);`;
    return code;
  }

  function generatePython(): string {
    if (mode === 'scan') {
      let code = `import boto3\n\n`;
      code += `table = boto3.resource('dynamodb').Table('${tableName}')\n`;
      const kwargs: string[] = [];
      if (limit) kwargs.push(`Limit=${limit}`);
      if (kwargs.length > 0) {
        code += `result = table.scan(${kwargs.join(', ')})\n`;
      } else {
        code += `result = table.scan()\n`;
      }
      code += `print(result['Items'])`;
      return code;
    }

    let code = `import boto3\nfrom boto3.dynamodb.conditions import Key\n\n`;
    code += `table = boto3.resource('dynamodb').Table('${tableName}')\n`;

    let keyCondition = `Key('${activeIndex.pk}').eq('${pkValue}')`;
    if (activeIndex.sk && skValue) {
      if (skOp === '=') keyCondition += ` & Key('${activeIndex.sk}').eq('${skValue}')`;
      else if (skOp === '<') keyCondition += ` & Key('${activeIndex.sk}').lt('${skValue}')`;
      else if (skOp === '>') keyCondition += ` & Key('${activeIndex.sk}').gt('${skValue}')`;
      else if (skOp === '<=') keyCondition += ` & Key('${activeIndex.sk}').lte('${skValue}')`;
      else if (skOp === '>=') keyCondition += ` & Key('${activeIndex.sk}').gte('${skValue}')`;
      else if (skOp === 'begins_with') keyCondition += ` & Key('${activeIndex.sk}').begins_with('${skValue}')`;
      else if (skOp === 'between') keyCondition += ` & Key('${activeIndex.sk}').between('${skValue}', '${skValue2}')`;
    }

    const kwargs: string[] = [`KeyConditionExpression=${keyCondition}`];
    if (indexName) kwargs.push(`IndexName='${indexName}'`);
    if (limit) kwargs.push(`Limit=${limit}`);

    code += `result = table.query(\n    ${kwargs.join(',\n    ')}\n)\n`;
    code += `print(result['Items'])`;
    return code;
  }

  function generateGo(): string {
    if (mode === 'scan') {
      let code = `package main\n\nimport (\n\t"context"\n\t"fmt"\n\n`;
      code += `\t"github.com/aws/aws-sdk-go-v2/aws"\n`;
      code += `\t"github.com/aws/aws-sdk-go-v2/config"\n`;
      code += `\t"github.com/aws/aws-sdk-go-v2/service/dynamodb"\n)\n\n`;
      code += `func main() {\n`;
      code += `\tcfg, _ := config.LoadDefaultConfig(context.TODO())\n`;
      code += `\tclient := dynamodb.NewFromConfig(cfg)\n\n`;
      code += `\tresult, err := client.Scan(context.TODO(), &dynamodb.ScanInput{\n`;
      code += `\t\tTableName: aws.String("${tableName}"),\n`;
      if (limit) code += `\t\tLimit: aws.Int32(${limit}),\n`;
      code += `\t})\n`;
      code += `\tif err != nil {\n\t\tpanic(err)\n\t}\n`;
      code += `\tfmt.Println(result.Items)\n}`;
      return code;
    }

    let code = `package main\n\nimport (\n\t"context"\n\t"fmt"\n\n`;
    code += `\t"github.com/aws/aws-sdk-go-v2/aws"\n`;
    code += `\t"github.com/aws/aws-sdk-go-v2/config"\n`;
    code += `\t"github.com/aws/aws-sdk-go-v2/service/dynamodb"\n`;
    code += `\t"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"\n)\n\n`;
    code += `func main() {\n`;
    code += `\tcfg, _ := config.LoadDefaultConfig(context.TODO())\n`;
    code += `\tclient := dynamodb.NewFromConfig(cfg)\n\n`;

    let keyExpr = `${activeIndex.pk} = :pk`;
    const exprValues: string[] = [];
    if (activePkType === 'N') {
      exprValues.push(`\t\t\t":pk": &types.AttributeValueMemberN{Value: "${pkValue}"}`);
    } else {
      exprValues.push(`\t\t\t":pk": &types.AttributeValueMemberS{Value: "${pkValue}"}`);
    }

    if (activeIndex.sk && skValue) {
      if (skOp === 'begins_with') {
        keyExpr += ` AND begins_with(${activeIndex.sk}, :sk)`;
      } else if (skOp === 'between') {
        keyExpr += ` AND ${activeIndex.sk} BETWEEN :sk AND :sk2`;
        if (activeSkType === 'N') {
          exprValues.push(`\t\t\t":sk2": &types.AttributeValueMemberN{Value: "${skValue2}"}`);
        } else {
          exprValues.push(`\t\t\t":sk2": &types.AttributeValueMemberS{Value: "${skValue2}"}`);
        }
      } else {
        keyExpr += ` AND ${activeIndex.sk} ${skOp} :sk`;
      }
      if (activeSkType === 'N') {
        exprValues.push(`\t\t\t":sk": &types.AttributeValueMemberN{Value: "${skValue}"}`);
      } else {
        exprValues.push(`\t\t\t":sk": &types.AttributeValueMemberS{Value: "${skValue}"}`);
      }
    }

    code += `\tresult, err := client.Query(context.TODO(), &dynamodb.QueryInput{\n`;
    code += `\t\tTableName:              aws.String("${tableName}"),\n`;
    if (indexName) code += `\t\tIndexName:              aws.String("${indexName}"),\n`;
    code += `\t\tKeyConditionExpression: aws.String("${keyExpr}"),\n`;
    code += `\t\tExpressionAttributeValues: map[string]types.AttributeValue{\n${exprValues.join(",\n")},\n\t\t},\n`;
    if (limit) code += `\t\tLimit: aws.Int32(${limit}),\n`;
    code += `\t})\n`;
    code += `\tif err != nil {\n\t\tpanic(err)\n\t}\n`;
    code += `\tfmt.Println(result.Items)\n}`;
    return code;
  }

  function buildJSFilter(filters: FilterCondition[]): { expr: string; names: string; values: string } | null {
    const valid = filters.filter(f => f.attribute && f.value);
    if (valid.length === 0) return null;
    const parts: string[] = [];
    const names: string[] = [];
    const values: string[] = [];
    valid.forEach((f, i) => {
      if (i > 0) parts.push(f.connector);
      const nk = `#fa${i}`;
      const vk = `:fv${i}`;
      names.push(`"${nk}": "${f.attribute}"`);
      values.push(`"${vk}": { "S": "${f.value}" }`);
      if (f.operator === 'begins_with') parts.push(`begins_with(${nk}, ${vk})`);
      else if (f.operator === 'between') {
        values.push(`"${vk}b": { "S": "${f.value2}" }`);
        parts.push(`${nk} BETWEEN ${vk} AND ${vk}b`);
      } else {
        parts.push(`${nk} ${f.operator} ${vk}`);
      }
    });
    return {
      expr: parts.join(' '),
      names: `{ ${names.join(', ')} }`,
      values: `{ ${values.join(', ')} }`,
    };
  }

  const code = useMemo(() => {
    switch (lang) {
      case 'javascript': return generateJS();
      case 'python': return generatePython();
      case 'go': return generateGo();
    }
  }, [lang, tableName, mode, pkValue, skOp, skValue, skValue2, indexName, filters, limit]);

  function handleCopy() {
    copyToClipboard(code);
    showToast('Code copied to clipboard');
  }

  return (
    <Modal title="Generate Code" size="lg" onClose={onClose}>
      <div class="flex items-center justify-between mb-4">
        <div class="ddb-codegen-tabs">
          <button
            class={`ddb-codegen-tab ${lang === 'javascript' ? 'active' : ''}`}
            onClick={() => setLang('javascript')}
          >JavaScript (SDK v3)</button>
          <button
            class={`ddb-codegen-tab ${lang === 'python' ? 'active' : ''}`}
            onClick={() => setLang('python')}
          >Python (boto3)</button>
          <button
            class={`ddb-codegen-tab ${lang === 'go' ? 'active' : ''}`}
            onClick={() => setLang('go')}
          >Go (aws-sdk-go-v2)</button>
        </div>
        <button class="btn btn-ghost btn-sm" onClick={handleCopy}>
          <CopyIcon /> Copy
        </button>
      </div>
      <div class="ddb-codegen-output">
        <pre class="ddb-codegen-pre"><code>{code}</code></pre>
      </div>
    </Modal>
  );
}
