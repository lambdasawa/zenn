import { useState } from "react";
import nunjucks from "nunjucks";

const someClientValue = "some client value";

async function callSendMailAPI(body) {
  await fetch("http://localhost:8192/", { method: "POST", body });
}

function fillTemplate(template) {
  return nunjucks.renderString(template, { clientValue: someClientValue });
}

function App() {
  const [template, setTemplate] = useState("<h1>Hello, {{ clientValue.toUpperCase() }}!</h1>");

  return (
    <div className="App">
      <textarea onChange={(e) => setTemplate(e.target.value)}>{template}</textarea>
      <button onClick={() => callSendMailAPI(fillTemplate(template))}>Send mail</button>
    </div>
  );
}

export default App;
