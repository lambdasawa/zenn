import { useState } from "react";

async function callSendMailAPI(body) {
  await fetch("http://localhost:8192/", { method: "POST", body });
}

function App() {
  const [template, setTemplate] = useState("<h1>Hello, {{ serverValue.toUpperCase() }}!</h1>");

  return (
    <div className="App">
      <textarea onChange={(e) => setTemplate(e.target.value)}>{template}</textarea>
      <button onClick={() => callSendMailAPI(template)}>Send mail</button>
    </div>
  );
}

export default App;
