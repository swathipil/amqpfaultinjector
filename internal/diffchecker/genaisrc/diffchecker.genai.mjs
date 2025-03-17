import fs from 'fs';

// NOTE FOR USER TO RUN THIS SCRIPT:
// > cd internal/diffchecker/genaisrc
// > npx genaiscript run diffchecker.genai.mjs

// Read the prompt from the markdown file
const prompt_file = 'prompts/amqpdiff.prompt.md';
const prompt = fs.readFileSync(prompt_file, 'utf-8');
const log_py = 'amqpproxy-traffic-python.jsonl';
const log_net = 'amqpproxy-traffic-net.jsonl';

script({
    model: "azure:gpt-4o",
})

// Prompting twice to see if this improves the results
$`${prompt}`
def("NET", fs.readFileSync(log_net, 'utf-8'))
def("PYTHON", fs.readFileSync(log_py, 'utf-8'))
$`${prompt}`.cacheControl("ephemeral")
//defDiff("DIFF", log_py, log_net)
