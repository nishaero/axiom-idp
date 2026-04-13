export default function Dashboard() {
  return (
    <div className="p-8">
      <h1 className="text-3xl font-bold mb-6 text-gray-900 dark:text-white">
        Welcome to Axiom IDP
      </h1>
      <p className="text-gray-600 dark:text-gray-300 mb-8">
        AI-Native Internal Developer Platform
      </p>
      <div className="grid grid-cols-3 gap-6">
        <div className="bg-white dark:bg-dark-800 p-6 rounded-lg shadow">
          <h3 className="font-semibold mb-2">Services</h3>
          <p className="text-2xl font-bold text-primary-600 dark:text-primary-400">0</p>
        </div>
        <div className="bg-white dark:bg-dark-800 p-6 rounded-lg shadow">
          <h3 className="font-semibold mb-2">Deployments</h3>
          <p className="text-2xl font-bold text-primary-600 dark:text-primary-400">0</p>
        </div>
        <div className="bg-white dark:bg-dark-800 p-6 rounded-lg shadow">
          <h3 className="font-semibold mb-2">Health</h3>
          <p className="text-2xl font-bold text-green-600 dark:text-green-400">OK</p>
        </div>
      </div>
    </div>
  )
}
