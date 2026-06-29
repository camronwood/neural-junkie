import React from 'react';

const App: React.FC = () => {
  const handleHelloWorld = () => {
    console.log('Hello, World!');
  };

  return (
    <div>
      <h1>Hello World App</h1>
      <button onClick={handleHelloWorld}>Say Hello</button>
    </div>
  );
};

export default App;